package mcpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/a3tai/library/internal/embedder"
	"github.com/a3tai/library/internal/library"
)

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id,omitempty"`
	Result  any            `json:"result,omitempty"`
	Error   *ResponseError `json:"error,omitempty"`
}

type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Server struct {
	store *library.Store
	embed *embedder.Provider // optional; nil disables vector path, falls back to FTS
}

// New constructs a Server backed by store. Vector search is disabled — use
// NewWithEmbedder to enable semantic search_passages.
func New(store *library.Store) *Server {
	return &Server{store: store}
}

// NewWithEmbedder constructs a Server that runs search_passages through
// the embeddings provider when LM Studio is reachable. The provider is a
// hot-swappable holder so the settings panel can change LM Studio details
// without restarting MCP. The FTS5 path remains the fallback when the
// provider is empty or LM Studio is offline.
func NewWithEmbedder(store *library.Store, emb *embedder.Provider) *Server {
	return &Server{store: store, embed: emb}
}

func (s *Server) ServeStdio(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 16*1024*1024)
	encoder := json.NewEncoder(out)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			encoder.Encode(Response{JSONRPC: "2.0", Error: &ResponseError{Code: -32700, Message: err.Error()}})
			continue
		}
		if req.ID == nil && strings.HasPrefix(req.Method, "notifications/") {
			continue
		}
		encoder.Encode(s.Handle(req))
	}
}

func (s *Server) HTTPHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ok", "server": "a3t-library-mcp"})
	})
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost")
		w.Header().Set("Access-Control-Allow-Headers", "authorization, content-type, x-a3t-mcp-token")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, Response{JSONRPC: "2.0", Error: &ResponseError{Code: -32700, Message: err.Error()}})
			return
		}
		writeJSON(w, s.Handle(req))
	})
	return mux
}

func (s *Server) Handle(req Request) Response {
	switch req.Method {
	case "initialize":
		return ok(req.ID, map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "a3t-library", "version": "1.0.0"},
		})
	case "ping":
		return ok(req.ID, map[string]any{})
	case "tools/list":
		return ok(req.ID, map[string]any{"tools": tools()})
	case "tools/call":
		return s.callTool(req)
	default:
		return fail(req.ID, -32601, "method not found")
	}
}

func (s *Server) callTool(req Request) Response {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return fail(req.ID, -32602, err.Error())
	}
	ctx := context.Background()
	switch params.Name {
	case "search_books":
		var args struct {
			Query string `json:"query"`
			Limit int    `json:"limit"`
		}
		decodeArgs(params.Arguments, &args)
		books, err := s.store.ListBooks(ctx, args.Query, args.Limit, 0)
		return toolResult(req.ID, books, err)
	case "search_passages":
		var args struct {
			Query  string `json:"query"`
			BookID string `json:"book_id"`
			Limit  int    `json:"limit"`
		}
		decodeArgs(params.Arguments, &args)
		passages, err := s.searchPassages(ctx, args.Query, args.BookID, args.Limit)
		return toolResult(req.ID, passages, err)
	case "get_book":
		var args struct {
			BookID string `json:"book_id"`
		}
		decodeArgs(params.Arguments, &args)
		book, err := s.store.GetBook(ctx, args.BookID)
		return toolResult(req.ID, book, err)
	case "get_passage":
		var args struct {
			PassageID string `json:"passage_id"`
		}
		decodeArgs(params.Arguments, &args)
		passage, err := s.store.GetPassage(ctx, args.PassageID)
		return toolResult(req.ID, passage, err)
	case "read_book_range":
		var args struct {
			BookID    string `json:"book_id"`
			FromChunk int    `json:"from_chunk"`
			Limit     int    `json:"limit"`
		}
		decodeArgs(params.Arguments, &args)
		if args.Limit <= 0 {
			args.Limit = 6
		}
		// Cap range reads so the model can't blow context with one
		// gigantic tool result. 12-passage pages × ~1800 chars ≈ ~22KB
		// max per call — still big, but a single call won't fill a
		// modest context window. Larger sweeps require multiple calls,
		// which the loop detector then watches for repetition.
		if args.Limit > 12 {
			args.Limit = 12
		}
		passages, err := s.store.BookPassages(ctx, args.BookID, args.FromChunk, args.Limit)
		return toolResult(req.ID, passages, err)
	case "list_books":
		var args struct {
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
		}
		decodeArgs(params.Arguments, &args)
		books, err := s.store.ListBooks(ctx, "", args.Limit, args.Offset)
		return toolResult(req.ID, books, err)
	default:
		return fail(req.ID, -32602, "unknown tool")
	}
}

// searchPassages routes through the vector path when an embedder is
// configured and the DB has at least one vector under that model. Falls
// back to FTS5 otherwise so the tool stays useful on fresh libraries or
// when LM Studio is offline.
func (s *Server) searchPassages(ctx context.Context, query, bookID string, limit int) ([]library.Passage, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return []library.Passage{}, nil
	}
	if client := s.embed.Get(); client != nil {
		model := client.Model
		has, err := s.store.HasAnyEmbeddings(ctx, model)
		if err == nil && has {
			embedCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			vec, err := client.EmbedOne(embedCtx, q)
			cancel()
			if err == nil {
				results, err := s.store.VectorSearchPassages(ctx, vec, model, bookID, limit)
				if err == nil {
					return results, nil
				}
				log.Printf("mcp vector search: %v", err)
			} else {
				log.Printf("mcp embed query: %v", err)
			}
		}
	}
	return s.store.SearchPassages(ctx, q, bookID, limit)
}

func decodeArgs(raw json.RawMessage, target any) {
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, target)
	}
}

func toolResult(id any, value any, err error) Response {
	if err != nil {
		return fail(id, -32000, err.Error())
	}
	payload, _ := json.MarshalIndent(value, "", "  ")
	return ok(id, map[string]any{"content": []map[string]string{{"type": "text", "text": string(payload)}}})
}

func ok(id any, result any) Response {
	return Response{JSONRPC: "2.0", ID: id, Result: result}
}

func fail(id any, code int, message string) Response {
	return Response{JSONRPC: "2.0", ID: id, Error: &ResponseError{Code: code, Message: message}}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

// Tools is the public accessor for the canonical tool list. Same shape we
// advertise over MCP, reused for OpenAI-style tool calling in the local
// chat experience.
func Tools() []map[string]any { return tools() }

func tools() []map[string]any {
	return []map[string]any{
		{
			"name":        "search_books",
			"description": "Search indexed book metadata by title, author, subject, and description.",
			"inputSchema": objectSchema(map[string]any{"query": stringSchema("Search query"), "limit": numberSchema("Maximum results")}, []string{"query"}),
		},
		{
			"name":        "search_passages",
			"description": "Search full-text passages from imported EPUB and text PDF files.",
			"inputSchema": objectSchema(map[string]any{"query": stringSchema("Search query"), "book_id": stringSchema("Optional book id"), "limit": numberSchema("Maximum results")}, []string{"query"}),
		},
		{
			"name":        "get_book",
			"description": "Return one book record by id.",
			"inputSchema": objectSchema(map[string]any{"book_id": stringSchema("Book id")}, []string{"book_id"}),
		},
		{
			"name":        "get_passage",
			"description": "Return one full passage by id.",
			"inputSchema": objectSchema(map[string]any{"passage_id": stringSchema("Passage id")}, []string{"passage_id"}),
		},
		{
			"name":        "read_book_range",
			"description": "Read a contiguous range of passages from a book in order. Use when you need the literal text of a section (a chapter, the opening, a specific chunk range) rather than searching by query. Returns up to `limit` passages starting at `from_chunk`. Default limit is 6 passages (~10KB); max is 12 per call. For larger reads, make several calls but be aware results add up quickly in context.",
			"inputSchema": objectSchema(map[string]any{
				"book_id":    stringSchema("Book id"),
				"from_chunk": numberSchema("Starting chunk_index (0 for the beginning of the book)"),
				"limit":      numberSchema("Maximum passages to return (default 6, capped at 12)"),
			}, []string{"book_id"}),
		},
		{
			"name":        "list_books",
			"description": "List recently imported books.",
			"inputSchema": objectSchema(map[string]any{"limit": numberSchema("Maximum results"), "offset": numberSchema("Offset")}, nil),
		},
	}
}

func objectSchema(properties map[string]any, required []string) map[string]any {
	if required == nil {
		required = []string{}
	}
	return map[string]any{"type": "object", "properties": properties, "required": required}
}

func stringSchema(description string) map[string]string {
	return map[string]string{"type": "string", "description": description}
}

func numberSchema(description string) map[string]string {
	return map[string]string{"type": "number", "description": description}
}
