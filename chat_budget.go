package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/a3tai/library/internal/aimeta"
)

func approxTokens(s string) int {
	if s == "" {
		return 0
	}
	return (len(s)*2 + 6) / 7 // ceil(len/3.5)
}

// messagesTokens is the total approx token cost of the message slice
// fed to ChatStream.
func messagesTokens(msgs []aimeta.Message) int {
	total := 0
	for _, m := range msgs {
		total += approxTokens(m.Content)
		total += approxTokens(m.Role) + approxTokens(m.Name) + approxTokens(m.ToolCallID)
		// Constant per-message overhead for the chat envelope.
		total += 6
		for _, tc := range m.ToolCalls {
			total += approxTokens(tc.Function.Name) + approxTokens(tc.Function.Arguments)
			total += 4
		}
	}
	return total
}

// compactMessagesToBudget is the chat's auto-compaction pass. Modeled on
// Claude-Code-style sliding-window-plus-summary: instead of dropping
// older turns when we run out of context, we collapse them into a single
// structured `[earlier conversation]` system message so the model still
// knows what it asked, what tools it used, and roughly what it learned.
//
// Stages applied in order until the budget is met:
//
//  1. Elide old tool results — already cheap, keeps recent tools verbatim.
//  2. Compact the "middle" (everything between the system prompt and the
//     last K=2 rounds) into a rules-based summary system message.
//  3. Shrink the recent-rounds window (K=1) and repeat the summary.
//  4. Trim the summary message itself to a hard cap.
//
// If after all stages we're still over budget the original tail still
// fits; the system prompt is the only thing that can push us over (the
// caller chose a too-small budget for the book metadata). We pass it
// through and let LM Studio surface the error.
//
// budgetTokens is the *total* prompt budget. Set it to roughly the
// model's context window minus the max_tokens reservation for the reply
// and a small safety margin.
//
// Returns a new slice; the input is not mutated.
func compactMessagesToBudget(msgs []aimeta.Message, budgetTokens int) (out []aimeta.Message, compacted bool) {
	if len(msgs) == 0 || budgetTokens <= 0 {
		return msgs, false
	}
	if messagesTokens(msgs) <= budgetTokens {
		return msgs, false
	}
	working := make([]aimeta.Message, len(msgs))
	copy(working, msgs)
	elideOldToolResults(working)
	if messagesTokens(working) <= budgetTokens {
		return working, false
	}

	systemIdx := -1
	for i, m := range working {
		if m.Role == "system" {
			systemIdx = i
			break
		}
	}
	if systemIdx < 0 {
		systemIdx = 0
	}

	// Try K=2 then K=1 recent-round windows. A "round" is bounded by user
	// messages: everything from one user message up to (but not including)
	// the next user message is one round.
	for _, keepLastRounds := range []int{2, 1, 0} {
		tailStart := tailStartForLastRounds(working, systemIdx, keepLastRounds)
		// Build candidate: system prompt + summary of middle + tail.
		candidate := make([]aimeta.Message, 0, 3+len(working)-tailStart)
		candidate = append(candidate, working[systemIdx])
		middle := working[systemIdx+1 : tailStart]
		if len(middle) > 0 {
			summary := summarizeHistorySegment(middle)
			candidate = append(candidate, summary)
		}
		candidate = append(candidate, working[tailStart:]...)

		if messagesTokens(candidate) <= budgetTokens {
			return candidate, true
		}
		// Still over — try shrinking the summary further before falling
		// through to a smaller window.
		if len(middle) > 0 {
			hard := hardCapSummary(candidate[1], budgetTokens-messagesTokens(append([]aimeta.Message{candidate[0]}, candidate[2:]...)))
			if hard.Content != "" {
				trimmed := make([]aimeta.Message, 0, len(candidate))
				trimmed = append(trimmed, candidate[0], hard)
				trimmed = append(trimmed, candidate[2:]...)
				if messagesTokens(trimmed) <= budgetTokens {
					return trimmed, true
				}
			}
		}
	}
	// Couldn't get under budget. Return our best attempt anyway —
	// system + last user message only — so the model at least has a
	// pointer to what the user actually asked.
	bestStart := tailStartForLastRounds(working, systemIdx, 0)
	candidate := []aimeta.Message{working[systemIdx]}
	if bestStart < len(working) {
		candidate = append(candidate, aimeta.Message{
			Role:    "system",
			Content: "[earlier conversation truncated to fit context window]",
		})
		candidate = append(candidate, working[bestStart:]...)
	}
	return candidate, true
}

// tailStartForLastRounds returns the index of the first message in the
// last `keepLastRounds` user-bounded rounds. If keepLastRounds is 0, it
// returns the index of the most recent user message (one-message "round").
// Returns len(msgs) if no user message exists (degenerate).
func tailStartForLastRounds(msgs []aimeta.Message, systemIdx int, keepLastRounds int) int {
	// Find user message indices after the system prompt.
	var userIdx []int
	for i := systemIdx + 1; i < len(msgs); i++ {
		if msgs[i].Role == "user" {
			userIdx = append(userIdx, i)
		}
	}
	if len(userIdx) == 0 {
		return len(msgs)
	}
	startUser := len(userIdx) - 1 - keepLastRounds
	if startUser < 0 {
		startUser = 0
	}
	return userIdx[startUser]
}

// summarizeHistorySegment produces a deterministic, rules-based summary
// of a slice of conversation history. The output is a single system
// message that names the user's prior asks, which tools fired, and the
// first/last lines of any assistant prose so the model knows what it
// said earlier. Intentionally compact (target a few hundred tokens) so
// repeated compactions stay cheap.
func summarizeHistorySegment(seg []aimeta.Message) aimeta.Message {
	var (
		userAsks    []string
		toolNames   = map[string]int{}
		assistText  []string
		toolSnippet []string
	)
	for _, m := range seg {
		switch m.Role {
		case "user":
			s := strings.TrimSpace(m.Content)
			if s != "" {
				userAsks = append(userAsks, truncateText(s, 160))
			}
		case "assistant":
			s := strings.TrimSpace(m.Content)
			if s != "" {
				assistText = append(assistText, truncateText(s, 240))
			}
			for _, tc := range m.ToolCalls {
				toolNames[tc.Function.Name]++
			}
		case "tool":
			s := strings.TrimSpace(m.Content)
			if s == "" || strings.HasPrefix(s, "[") && strings.Contains(s, "elided]") {
				continue
			}
			toolSnippet = append(toolSnippet, fmt.Sprintf("%s → %s", fallback(m.Name, "tool"), truncateText(s, 200)))
		}
	}

	var b strings.Builder
	b.WriteString("[earlier conversation summary]\n")
	if len(userAsks) > 0 {
		b.WriteString("User previously asked:\n")
		for _, q := range userAsks {
			fmt.Fprintf(&b, "- %s\n", q)
		}
	}
	if len(toolNames) > 0 {
		b.WriteString("Tools used: ")
		parts := []string{}
		for name, n := range toolNames {
			parts = append(parts, fmt.Sprintf("%s×%d", name, n))
		}
		b.WriteString(strings.Join(parts, ", "))
		b.WriteString("\n")
	}
	if len(toolSnippet) > 0 {
		b.WriteString("Notable tool findings:\n")
		// Keep the most recent 3 findings — older ones already faded.
		start := 0
		if len(toolSnippet) > 3 {
			start = len(toolSnippet) - 3
		}
		for _, s := range toolSnippet[start:] {
			fmt.Fprintf(&b, "- %s\n", s)
		}
	}
	if len(assistText) > 0 {
		b.WriteString("Assistant previously said:\n")
		// First and last meaningful turn.
		fmt.Fprintf(&b, "- (first) %s\n", assistText[0])
		if len(assistText) > 1 {
			fmt.Fprintf(&b, "- (most recent) %s\n", assistText[len(assistText)-1])
		}
	}
	b.WriteString("Continue from this context; the user's latest message follows.")
	return aimeta.Message{Role: "system", Content: b.String()}
}

// hardCapSummary forcibly truncates a summary message to fit inside a
// remaining budget. Used as the last-ditch shrink before falling back
// to a smaller recent-rounds window.
func hardCapSummary(summary aimeta.Message, budgetTokens int) aimeta.Message {
	if budgetTokens <= 0 {
		return aimeta.Message{}
	}
	// Allow ~3.5 bytes per token (matches approxTokens). Leave a margin.
	maxBytes := (budgetTokens * 7) / 2
	if maxBytes < 200 {
		maxBytes = 200
	}
	if len(summary.Content) <= maxBytes {
		return summary
	}
	return aimeta.Message{
		Role:    "system",
		Content: summary.Content[:maxBytes-15] + "\n[summary cut]",
	}
}

// resolveChatBudget computes the prompt token budget for the configured
// chat model. We query LM Studio's /api/v0/models for the loaded
// context_length when available; failing that we fall back to a
// conservative 6000-token default. The reply budget (max_tokens) and a
// safety margin are subtracted so the prompt fits inside the window with
// room for the response.
//
// Results are cached per (model, baseURL) to avoid hitting LM Studio on
// every turn. Cache invalidates when the settings panel rebuilds the AI
// client (handled implicitly: the lookup is run against the current
// client, and a new client = new cache miss).
const (
	defaultChatBudgetTokens = 6000
	chatReplyBudgetTokens   = 1024
	chatSafetyMarginTokens  = 384
)

func (s *LibraryService) resolveChatBudget(ctx context.Context, model string) int {
	s.chatBudgetMu.Lock()
	defer s.chatBudgetMu.Unlock()
	if s.chatBudgetCache == nil {
		s.chatBudgetCache = map[string]int{}
	}
	key := s.ai.BaseURL + "|" + model
	if v, ok := s.chatBudgetCache[key]; ok {
		return v
	}
	budget := defaultChatBudgetTokens
	if model != "" {
		probeCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
		info, err := s.ai.LookupModel(probeCtx, model)
		cancel()
		if err == nil && info.ContextLength > 0 {
			usable := info.ContextLength - chatReplyBudgetTokens - chatSafetyMarginTokens
			if usable > 1000 {
				budget = usable
			}
		}
	}
	s.chatBudgetCache[key] = budget
	return budget
}

// isContextError matches the various wordings LM Studio (and downstream
// llama.cpp servers) use to signal a context-window overflow. Kept as a
// single substring whitelist so a new wording shows up as "unknown
// error" rather than silently triggering aggressive retries on unrelated
// failures.
func isContextError(err error) bool {
	if err == nil {
		return false
	}
	low := strings.ToLower(err.Error())
	switch {
	case strings.Contains(low, "context size"),
		strings.Contains(low, "context length"),
		strings.Contains(low, "context window"),
		strings.Contains(low, "exceeds the context"),
		strings.Contains(low, "too long for"),
		strings.Contains(low, "n_ctx"):
		return true
	}
	return false
}

// friendlyChatError rewrites LM Studio's verbose error responses into a
// shorter, action-oriented message for the chat UI. Logs the raw error
// separately so debug detail isn't lost.
func friendlyChatError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	low := strings.ToLower(msg)
	switch {
	case strings.Contains(low, "context size") || strings.Contains(low, "context length") || strings.Contains(low, "context window"):
		return "LM Studio reports context size exceeded — try a shorter question, a fresh chat, or load a chat model with a larger context window."
	case strings.Contains(low, "no models loaded"):
		return "No chat model is loaded in LM Studio. Load a model and try again."
	case strings.Contains(low, "connection refused") || strings.Contains(low, "no such host"):
		return "Can't reach LM Studio. Is the local server running?"
	default:
		return msg
	}
}
