// Package embedder talks to a local OpenAI-compatible embeddings endpoint —
// LM Studio by default. It exposes a single Embed call that returns dense
// float32 vectors for an arbitrary batch of input strings.
//
// Like aimeta, every operation is best-effort: if the endpoint is down the
// caller gets a clean error and the rest of the app degrades to lexical
// search.
package embedder

import (
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
	// Default embedding model identifier. Targets SuperPauly's harrier
	// embedding model (huggingface.co/SuperPauly/harrier-oss-v1-0.6b-gguf)
	// loaded into LM Studio. LM Studio routes the request to whichever
	// embedding model is loaded; matching this to the loaded model's id
	// avoids ambiguity when multiple models are available. Override via
	// LIBRARY_LMSTUDIO_EMBED_MODEL if your LM Studio surfaces it under a
	// different identifier (e.g. the full HF repo path).
	defaultModel = "harrier-oss-v1-0.6b"
)

// Setting keys this package reads out of the library `settings` table. The
// keys are part of the public contract between the settings UI and the
// backend — keep these stable across releases.
const (
	SettingURL        = "lmstudio.url"
	SettingAPIKey     = "lmstudio.api_key"
	SettingEmbedModel = "lmstudio.embed_model"
)

// Client is an OpenAI-compatible embeddings client.
type Client struct {
	BaseURL string
	Model   string
	APIKey  string
	HTTP    *http.Client
}

// Config carries the three knobs the user can tune. Build with Resolve and
// pass to NewFromConfig — that's the canonical path. New() exists only for
// callers that have no access to stored settings yet (cmd/librarymcp).
type Config struct {
	BaseURL string
	Model   string
	APIKey  string
}

// New reads configuration from the environment and built-in defaults. Used
// by callers that don't have a settings store handy. The desktop app should
// use Resolve+NewFromConfig instead so DB-backed user settings win.
func New() *Client {
	return NewFromConfig(Resolve(nil))
}

// NewFromConfig builds a client from explicit values. Empty fields fall
// back to env vars and then built-in defaults so a partial Config still
// produces a working client.
func NewFromConfig(cfg Config) *Client {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = envDefault(defaultBaseURL, "LIBRARY_LMSTUDIO_URL")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = envDefault(defaultModel, "LIBRARY_LMSTUDIO_EMBED_MODEL")
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
// after the DB lookup is filled in from env + defaults inside
// NewFromConfig. Pass `nil` for stored to get the env-only path.
func Resolve(stored map[string]string) Config {
	get := func(k string) string { return strings.TrimSpace(stored[k]) }
	return Config{
		BaseURL: get(SettingURL),
		Model:   get(SettingEmbedModel),
		APIKey:  get(SettingAPIKey),
	}
}

// ListModels returns the model ids LM Studio advertises on /v1/models.
// Used by the settings panel to populate the model dropdowns. The returned
// slice may include chat AND embedding models — LM Studio doesn't tag them
// by type in the OpenAI-compatible response — so callers should treat it
// as a single combined pool.
func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	if c == nil {
		return nil, errors.New("embedder: nil client")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
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
		return nil, fmt.Errorf("LM Studio /models: %s", strings.TrimSpace(string(data)))
	}
	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("decode /models: %w", err)
	}
	if parsed.Error != nil {
		return nil, errors.New(parsed.Error.Message)
	}
	out := make([]string, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		if m.ID != "" {
			out = append(out, m.ID)
		}
	}
	return out, nil
}

// Available returns true if /models responds within ~1.5s. Used to decide
// whether vector search is currently usable; if not we fall back to FTS.
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

type embedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embedResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Embed returns one vector per input string, in the same order. Empty inputs
// are passed through as zero vectors to keep the slice aligned with the
// caller's passage list — callers should filter those out before storing.
//
// The endpoint is called once per Embed invocation; chunk the inputs at the
// call site if you need to respect a max-batch limit.
func (c *Client) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if c == nil {
		return nil, errors.New("embedder: nil client")
	}
	if len(inputs) == 0 {
		return nil, nil
	}
	// Reject control characters that LM Studio's tokenizer trips on. The
	// importer already strips most of these but be defensive.
	cleaned := make([]string, len(inputs))
	for i, s := range inputs {
		cleaned[i] = strings.ReplaceAll(s, "\x00", "")
	}
	body := embedRequest{Model: c.Model, Input: cleaned}
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/embeddings", bytes.NewReader(buf))
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
		return nil, fmt.Errorf("LM Studio /embeddings: %s", strings.TrimSpace(string(data)))
	}
	var parsed embedResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("decode embeddings: %w", err)
	}
	if parsed.Error != nil {
		return nil, errors.New(parsed.Error.Message)
	}
	if len(parsed.Data) != len(inputs) {
		return nil, fmt.Errorf("LM Studio returned %d vectors for %d inputs", len(parsed.Data), len(inputs))
	}
	out := make([][]float32, len(inputs))
	for _, item := range parsed.Data {
		if item.Index < 0 || item.Index >= len(inputs) {
			return nil, fmt.Errorf("embedding index %d out of range", item.Index)
		}
		out[item.Index] = item.Embedding
	}
	for i, v := range out {
		if v == nil {
			return nil, fmt.Errorf("missing vector for input %d", i)
		}
	}
	return out, nil
}

// EmbedOne is a convenience wrapper for the common single-string case (e.g.
// the user's search query).
func (c *Client) EmbedOne(ctx context.Context, input string) ([]float32, error) {
	vecs, err := c.Embed(ctx, []string{input})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, errors.New("embedder: no vector returned")
	}
	return vecs[0], nil
}

func (c *Client) applyAuth(req *http.Request) {
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
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
