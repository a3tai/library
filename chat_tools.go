package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/a3tai/library/internal/aimeta"
	"github.com/a3tai/library/internal/library"
	"github.com/a3tai/library/internal/mcpserver"
)

func chatSystemPrompt(book library.Book, toc []library.TOCEntry) string {
	var b strings.Builder
	b.WriteString("You are a careful reading assistant grounded in a single book the user is currently viewing.\n")
	b.WriteString("Use the metadata below as primary context. When you need to quote or verify text, call the search_passages tool with the current book_id.\n\n")

	b.WriteString("=== CURRENT BOOK ===\n")
	fmt.Fprintf(&b, "Title: %s\n", book.Title)
	fmt.Fprintf(&b, "Authors: %s\n", fallback(book.Authors, "Unknown"))
	if book.Publisher != "" {
		fmt.Fprintf(&b, "Publisher: %s\n", book.Publisher)
	}
	if book.PublishedDate != "" {
		fmt.Fprintf(&b, "Published: %s\n", book.PublishedDate)
	}
	if book.ISBN13 != "" {
		fmt.Fprintf(&b, "ISBN: %s\n", book.ISBN13)
	} else if book.ISBN10 != "" {
		fmt.Fprintf(&b, "ISBN: %s\n", book.ISBN10)
	}
	if book.Language != "" {
		fmt.Fprintf(&b, "Language: %s\n", book.Language)
	}
	fmt.Fprintf(&b, "Format: %s\n", book.Format)
	fmt.Fprintf(&b, "Index status: %s\n", book.IndexStatus)
	fmt.Fprintf(&b, "Passages indexed: %d\n", book.PassageCount)
	fmt.Fprintf(&b, "Book ID (for tool calls): %s\n", book.ID)

	if desc := strings.TrimSpace(book.Description); desc != "" {
		b.WriteString("\nDescription / metadata:\n")
		b.WriteString(truncateText(desc, 800))
		b.WriteString("\n")
	}

	if len(toc) > 0 {
		b.WriteString("\nTable of contents (chunkIndex → label, ~pages):\n")
		limit := len(toc)
		if limit > 30 {
			limit = 30
		}
		for i := 0; i < limit; i++ {
			e := toc[i]
			fmt.Fprintf(&b, "  %d → %s (%d)\n", e.ChunkIndex, e.Label, e.Pages)
		}
		if len(toc) > limit {
			fmt.Fprintf(&b, "  … %d more sections\n", len(toc)-limit)
		}
	}

	b.WriteString("\n=== TOOL USAGE ===\n")
	b.WriteString("- search_passages with book_id=" + book.ID + " is the right tool when the user asks about a topic — it returns the most relevant passages by query.\n")
	b.WriteString("- read_book_range with book_id=" + book.ID + " is the right tool when the user asks about a specific *section* of the book (a chapter, the opening, a particular chunk range). It returns consecutive passages in order. Use the TOC above to choose a starting chunk_index. Default is 6 passages per call.\n")
	b.WriteString("- get_passage returns one passage by id when you need a single passage's full text.\n")
	b.WriteString("- Other tools (search_books, list_books, get_book) operate over the user's whole library — use them for cross-book questions.\n")
	b.WriteString("- Prefer reading directly with read_book_range when the user asks 'what does the book say about X' and you already know roughly where to look from the TOC. Use search_passages when the location is unknown.\n")
	b.WriteString("\nWhen to STOP calling tools:\n")
	b.WriteString("- If a search_passages call returns [] (empty), do NOT repeat the same query. Either reformulate, switch to read_book_range, or answer from what you already have.\n")
	b.WriteString("- If you have already called read_book_range twice in this turn, answer from the passages you have rather than fetching more — the user is waiting.\n")
	b.WriteString("- If the user's question is a follow-up asking for more detail or clarification (e.g. 'tell me more', 'that's not enough', 'be more specific'), ELABORATE on the information you already gathered in previous turns. Do NOT re-search the same ground; the user wants a richer answer from existing context.\n")
	b.WriteString("- The metadata above (title, authors, description, TOC) is authoritative for high-level questions about what the book is. Use it directly rather than calling tools when the answer is already visible.\n")
	b.WriteString("\nQuote sparingly; cite passage labels when available. If the answer isn't supported by the text or metadata above, say so plainly rather than guessing.")
	return b.String()
}

func buildOpenAITools() []aimeta.Tool {
	raw := mcpserver.Tools()
	tools := make([]aimeta.Tool, 0, len(raw))
	for _, t := range raw {
		var ai aimeta.Tool
		ai.Type = "function"
		if name, ok := t["name"].(string); ok {
			ai.Function.Name = name
		}
		if desc, ok := t["description"].(string); ok {
			ai.Function.Description = desc
		}
		if schema, ok := t["inputSchema"].(map[string]any); ok {
			ai.Function.Parameters = schema
		}
		tools = append(tools, ai)
	}
	return tools
}

func executeMCPTool(server *mcpserver.Server, call aimeta.ToolCall) (string, error) {
	args := json.RawMessage(call.Function.Arguments)
	if len(args) == 0 || string(args) == `""` {
		args = json.RawMessage(`{}`)
	}
	params, _ := json.Marshal(map[string]any{
		"name":      call.Function.Name,
		"arguments": args,
	})
	resp := server.Handle(mcpserver.Request{
		JSONRPC: "2.0",
		ID:      call.ID,
		Method:  "tools/call",
		Params:  params,
	})
	if resp.Error != nil {
		return "", fmt.Errorf("%s: %s", call.Function.Name, resp.Error.Message)
	}
	// MCP returns {content: [{type: "text", text: "..."}]}; flatten.
	if m, ok := resp.Result.(map[string]any); ok {
		if items, ok := m["content"].([]map[string]string); ok && len(items) > 0 {
			return items[0]["text"], nil
		}
	}
	out, _ := json.Marshal(resp.Result)
	return string(out), nil
}

func toAI(m ChatMessage) aimeta.Message {
	out := aimeta.Message{Role: m.Role, Content: m.Content, Name: m.Name, ToolCallID: m.ToolCallID}
	for _, t := range m.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, aimeta.ToolCall{
			ID:   t.ID,
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{Name: t.Name, Arguments: t.Arguments},
		})
	}
	return out
}

func fromAI(m aimeta.Message) ChatMessage {
	out := ChatMessage{Role: m.Role, Content: m.Content, Name: m.Name, ToolCallID: m.ToolCallID}
	for _, t := range m.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, ChatToolCall{
			ID:        t.ID,
			Name:      t.Function.Name,
			Arguments: t.Function.Arguments,
		})
	}
	return out
}

func fallback(s, fb string) string {
	if strings.TrimSpace(s) == "" {
		return fb
	}
	return s
}

func iff(cond bool, s string) string {
	if cond {
		return s
	}
	return ""
}

// elideOldToolResults rewrites the content of tool messages that aren't part
// of the most recent contiguous batch. The LLM only needs the latest batch
// in full to decide its next move; older batches keep their position +
// tool_call_id (so the chat shape stays valid) but their bodies shrink to
// a one-line placeholder.
func elideOldToolResults(messages []aimeta.Message) {
	// Walk back to find the start of the most recent batch of tool messages.
	last := -1
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "tool" {
			last = i
			break
		}
	}
	if last < 0 {
		return
	}
	batchStart := last
	for batchStart > 0 && messages[batchStart-1].Role == "tool" {
		batchStart--
	}
	for i := 0; i < batchStart; i++ {
		if messages[i].Role != "tool" {
			continue
		}
		name := messages[i].Name
		if name == "" {
			name = "tool"
		}
		messages[i].Content = "[" + name + " result elided]"
	}
}

func truncateText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n\n[truncated]"
}
