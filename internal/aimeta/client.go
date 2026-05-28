// Package aimeta talks to a local OpenAI-compatible inference endpoint —
// LM Studio by default. It exposes two flavours of work: extracting
// structured metadata from a book's sample passages, and a single chat turn
// (with optional tool calling) so the frontend can chat with a book backed
// by our MCP tools.
//
// All operations are best-effort: if the endpoint is down or returns
// nonsense, callers receive a clean error and continue without the AI
// fields.
package aimeta

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultBaseURL = "http://127.0.0.1:1234/v1"
	defaultModel   = "lmstudio-community" // LM Studio uses whatever model is loaded; the literal value is mostly ignored
)

// Setting keys this package reads out of the library `settings` table.
// Shared with the settings UI; keep stable across releases.
const (
	SettingURL       = "lmstudio.url"
	SettingAPIKey    = "lmstudio.api_key"
	SettingChatModel = "lmstudio.chat_model"
)

// Client wraps an OpenAI-compatible chat-completions endpoint.
type Client struct {
	BaseURL string
	Model   string
	APIKey  string
	HTTP    *http.Client
}

// Config is the explicit-values constructor input. Empty fields fall back
// to env vars + defaults inside NewFromConfig.
type Config struct {
	BaseURL string
	Model   string
	APIKey  string
}

// New returns a Client wired from environment vars + built-in defaults.
// Callers with a settings store should prefer Resolve+NewFromConfig.
//
//	LIBRARY_LMSTUDIO_URL      — base URL ending in /v1 (default 127.0.0.1:1234/v1)
//	LIBRARY_LMSTUDIO_MODEL    — model name to send (LM Studio routes by loaded model)
//	LIBRARY_LMSTUDIO_API_KEY  — bearer token if LM Studio's auth is enabled.
//	                          LM_API_TOKEN and OPENAI_API_KEY are read as fallbacks.
func New() *Client {
	return NewFromConfig(Resolve(nil))
}

// NewFromConfig builds a Client from explicit values. Empty fields fall
// back to env then built-in defaults so a partially-populated Config is
// safe to pass.
func NewFromConfig(cfg Config) *Client {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = envDefault(defaultBaseURL, "LIBRARY_LMSTUDIO_URL")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = envDefault(defaultModel, "LIBRARY_LMSTUDIO_MODEL")
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		cfg.APIKey = firstEnv("LIBRARY_LMSTUDIO_API_KEY", "LM_API_TOKEN", "OPENAI_API_KEY")
	}
	return &Client{
		BaseURL: strings.TrimRight(cfg.BaseURL, "/"),
		Model:   cfg.Model,
		APIKey:  cfg.APIKey,
		HTTP:    &http.Client{Timeout: 120 * time.Second},
	}
}

// Resolve folds DB-stored settings into a Config. Anything left blank
// becomes env / default inside NewFromConfig. Pass nil for the env-only
// path.
func Resolve(stored map[string]string) Config {
	get := func(k string) string { return strings.TrimSpace(stored[k]) }
	return Config{
		BaseURL: get(SettingURL),
		Model:   get(SettingChatModel),
		APIKey:  get(SettingAPIKey),
	}
}

func envDefault(fallback string, keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return fallback
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

// ModelInfo captures the subset of LM Studio's /api/v0/models response
// that we care about for context-management decisions. LM Studio surfaces
// the loaded context window per model under a vendor-specific extension
// to the OpenAI schema (the field name varies across versions, so we
// accept all the spellings we've seen and pick the first non-zero one).
type ModelInfo struct {
	ID            string
	ContextLength int
}

type modelInfoRaw struct {
	ID                  string `json:"id"`
	MaxContextLength    int    `json:"max_context_length"`
	LoadedContextLength int    `json:"loaded_context_length"`
	ContextLength       int    `json:"context_length"`
}

func (m modelInfoRaw) ctx() int {
	if m.LoadedContextLength > 0 {
		return m.LoadedContextLength
	}
	if m.MaxContextLength > 0 {
		return m.MaxContextLength
	}
	return m.ContextLength
}

// LookupModel queries LM Studio's extended /api/v0/models endpoint for
// per-model metadata (context window primarily). Falls back to the
// OpenAI-compatible /v1/models which doesn't include context size, in
// which case ContextLength is 0 and the caller should use a conservative
// default. Best-effort — errors are returned but callers should treat
// them as "unknown" rather than fatal.
func (c *Client) LookupModel(ctx context.Context, id string) (ModelInfo, error) {
	if c == nil || id == "" {
		return ModelInfo{ID: id}, errors.New("aimeta: nil client or empty id")
	}
	// LM Studio's extended endpoint is at /api/v0/models — we have to
	// strip the trailing /v1 to reach it.
	base := strings.TrimRight(strings.TrimSuffix(c.BaseURL, "/v1"), "/")
	url := base + "/api/v0/models/" + id
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ModelInfo{ID: id}, err
	}
	c.applyAuth(req)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return ModelInfo{ID: id}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		// Endpoint not supported by this LM Studio version.
		return ModelInfo{ID: id}, fmt.Errorf("LM Studio /api/v0/models/%s: status %d", id, res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return ModelInfo{ID: id}, err
	}
	var raw modelInfoRaw
	if err := json.Unmarshal(body, &raw); err != nil {
		return ModelInfo{ID: id}, err
	}
	return ModelInfo{ID: id, ContextLength: raw.ctx()}, nil
}

// Available reports whether the configured endpoint responds. Used to gate
// AI-dependent flows so the rest of the app degrades gracefully when the
// user hasn't started LM Studio.
func (c *Client) Available(ctx context.Context) bool {
	if c == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/models", nil)
	if err != nil {
		return false
	}
	c.applyAuth(req)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	return res.StatusCode >= 200 && res.StatusCode < 500
}

func (c *Client) applyAuth(req *http.Request) {
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
}

// ---- Chat (single turn) ----------------------------------------------------

// Tool mirrors the OpenAI function-tool schema. Use FromMCP to convert
// MCP-style tool definitions.
type Tool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string         `json:"name"`
		Description string         `json:"description,omitempty"`
		Parameters  map[string]any `json:"parameters,omitempty"`
	} `json:"function"`
}

// Message is one turn in a chat completion request.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall is the model's request to invoke a tool.
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// ChatResponse is what we surface from one /chat/completions round-trip.
type ChatResponse struct {
	Message      Message `json:"message"`
	FinishReason string  `json:"finishReason"`
	Model        string  `json:"model"`
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	ToolChoice  any       `json:"tool_choice,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream"`
}

type chatRawResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Chat performs a single chat completion. Pass tools to enable function
// calling; pass nil to disable.
func (c *Client) Chat(ctx context.Context, messages []Message, tools []Tool) (ChatResponse, error) {
	body := chatRequest{
		Model:       c.Model,
		Messages:    messages,
		Tools:       tools,
		Temperature: 0.4,
		MaxTokens:   1024,
		Stream:      false,
	}
	if len(tools) > 0 {
		body.ToolChoice = "auto"
	}
	raw, err := c.postJSON(ctx, "/chat/completions", body)
	if err != nil {
		return ChatResponse{}, err
	}
	var parsed chatRawResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return ChatResponse{}, fmt.Errorf("decode chat response: %w", err)
	}
	if parsed.Error != nil {
		return ChatResponse{}, errors.New(parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return ChatResponse{}, errors.New("no choices in chat response")
	}
	return ChatResponse{
		Message:      parsed.Choices[0].Message,
		FinishReason: parsed.Choices[0].FinishReason,
		Model:        parsed.Model,
	}, nil
}

// DeltaCallback is invoked once per content fragment as the stream arrives.
// Tool-call fragments are buffered server-side and surface only in the final
// assembled Message returned by ChatStream.
type DeltaCallback func(contentDelta string)

// ChatStream is the streaming flavour of Chat. It opens the OpenAI-style SSE
// endpoint, calls onDelta for each text fragment, and returns the assembled
// ChatResponse (content + tool_calls + model + finish_reason) once the stream
// closes. onDelta may be nil if the caller only wants the final assembly.
func (c *Client) ChatStream(ctx context.Context, messages []Message, tools []Tool, onDelta DeltaCallback) (ChatResponse, error) {
	body := chatRequest{
		Model:       c.Model,
		Messages:    messages,
		Tools:       tools,
		Temperature: 0.4,
		MaxTokens:   1024,
		Stream:      true,
	}
	if len(tools) > 0 {
		body.ToolChoice = "auto"
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(buf))
	if err != nil {
		return ChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	c.applyAuth(req)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("LM Studio request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		data, _ := io.ReadAll(res.Body)
		return ChatResponse{}, fmt.Errorf("LM Studio /chat/completions: %s", strings.TrimSpace(string(data)))
	}

	// tool_calls arrive fragmented across deltas, indexed by an `index` field.
	// Accumulate each one piece by piece, then flatten in order at the end.
	type toolBuf struct {
		ID, Type, Name string
		Args           strings.Builder
	}
	bufs := map[int]*toolBuf{}
	var order []int
	var content strings.Builder
	model := c.Model
	finishReason := ""

	scanner := bufio.NewScanner(res.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}
		var chunk struct {
			Model   string `json:"model"`
			Choices []struct {
				Delta struct {
					Role      string `json:"role"`
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error,omitempty"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			// Some servers occasionally emit non-JSON keepalive frames; skip them.
			continue
		}
		if chunk.Error != nil {
			return ChatResponse{}, errors.New(chunk.Error.Message)
		}
		if chunk.Model != "" {
			model = chunk.Model
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		ch := chunk.Choices[0]
		if ch.Delta.Content != "" {
			content.WriteString(ch.Delta.Content)
			if onDelta != nil {
				onDelta(ch.Delta.Content)
			}
		}
		for _, tc := range ch.Delta.ToolCalls {
			b, ok := bufs[tc.Index]
			if !ok {
				b = &toolBuf{}
				bufs[tc.Index] = b
				order = append(order, tc.Index)
			}
			if tc.ID != "" {
				b.ID = tc.ID
			}
			if tc.Type != "" {
				b.Type = tc.Type
			}
			if tc.Function.Name != "" {
				b.Name = tc.Function.Name
			}
			if tc.Function.Arguments != "" {
				b.Args.WriteString(tc.Function.Arguments)
			}
		}
		if ch.FinishReason != "" {
			finishReason = ch.FinishReason
		}
	}
	if err := scanner.Err(); err != nil {
		return ChatResponse{}, fmt.Errorf("read SSE stream: %w", err)
	}

	assembled := Message{Role: "assistant", Content: content.String()}
	for _, idx := range order {
		b := bufs[idx]
		call := ToolCall{ID: b.ID, Type: b.Type}
		if call.Type == "" {
			call.Type = "function"
		}
		call.Function.Name = b.Name
		call.Function.Arguments = b.Args.String()
		assembled.ToolCalls = append(assembled.ToolCalls, call)
	}
	return ChatResponse{Message: assembled, FinishReason: finishReason, Model: model}, nil
}

// ---- Metadata extraction --------------------------------------------------

// Metadata is the structured payload we ask the model to fill in for a book.
// All fields are optional; the caller decides what to persist. Categories are
// broad BISAC-style top-level buckets; Subjects are finer-grained topics.
type Metadata struct {
	Categories []string `json:"categories"`
	Subjects   []string `json:"subjects"`
	Summary    string   `json:"summary"`
	Genre      string   `json:"genre"`
	Era        string   `json:"era"`
	Audience   string   `json:"audience"`
}

// ExtractMetadata feeds a few sample passages plus what we already know about
// a book into the model and asks it to return a small JSON object. The model
// sees a strict instruction to emit JSON only; we still defensively pull the
// first JSON object out of the response.
func (c *Client) ExtractMetadata(ctx context.Context, title, authors string, samples []string) (Metadata, error) {
	if len(samples) == 0 {
		return Metadata{}, errors.New("no sample passages provided")
	}
	context := strings.Join(samples, "\n\n---\n\n")
	if len(context) > 6000 {
		context = context[:6000]
	}
	user := fmt.Sprintf(`Title: %s
Authors: %s

Below are excerpts from the book. Using ONLY these excerpts, return a strict
JSON object describing the work. Keep each field short and grounded in the
text — leave a field empty (or its array empty) if the excerpts don't support it.

Definitions (be strict):
- "categories": 1–4 BROAD top-level buckets, BISAC-style. Examples:
  "Computers", "Fiction", "History", "Philosophy", "Business",
  "Mathematics", "Biography & Autobiography". NOT specific topics.
- "subjects": 4–12 SPECIFIC topics covered in the book — concrete concepts,
  techniques, places, people, or ideas drawn from the excerpts. Examples:
  "halting problem", "Turing machines", "Vienna Circle", "tax law in
  Ming-dynasty China". Each subject should be 1–4 words.
- "summary": 1–3 sentences describing what the book is about.
- "genre": one short label (e.g. "textbook", "novel", "essay collection").
- "era": time period the book is about, if any (e.g. "20th century",
  "ancient Rome"); empty for timeless works.
- "audience": e.g. "graduate students", "general readers".

Excerpts:
%s

Return JSON with this exact shape and nothing else:
{
  "categories": ["..."],
  "subjects": ["..."],
  "summary": "...",
  "genre": "...",
  "era": "...",
  "audience": "..."
}`, title, authors, context)

	resp, err := c.Chat(ctx, []Message{
		{Role: "system", Content: "You extract structured metadata for a local library. Reply with a single JSON object and nothing else."},
		{Role: "user", Content: user},
	}, nil)
	if err != nil {
		return Metadata{}, err
	}
	jsonText := firstJSONObject(resp.Message.Content)
	if jsonText == "" {
		return Metadata{}, fmt.Errorf("model returned no JSON: %q", truncate(resp.Message.Content, 200))
	}
	var meta Metadata
	if err := json.Unmarshal([]byte(jsonText), &meta); err != nil {
		return Metadata{}, fmt.Errorf("parse metadata: %w", err)
	}
	return meta, nil
}

// ---- internals ------------------------------------------------------------

func (c *Client) postJSON(ctx context.Context, path string, body any) ([]byte, error) {
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	c.applyAuth(req)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("LM Studio request: %w", err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("LM Studio %s %s: %s", req.Method, path, strings.TrimSpace(string(data)))
	}
	return data, nil
}

// firstJSONObject extracts the first balanced { ... } substring from s. The
// model occasionally wraps the JSON in prose or code fences; this function
// tolerates either.
func firstJSONObject(s string) string {
	depth := 0
	start := -1
	for i, r := range s {
		switch r {
		case '{':
			if depth == 0 {
				start = i
			}
			depth++
		case '}':
			if depth > 0 {
				depth--
				if depth == 0 && start >= 0 {
					return s[start : i+1]
				}
			}
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
