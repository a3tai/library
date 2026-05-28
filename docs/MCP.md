# MCP

A3T: Library exposes the local SQLite index through MCP so AI clients can search the user's library without copying the database elsewhere.

## Transports

Standalone stdio server:

```sh
go run ./cmd/librarymcp
```

Specific database:

```sh
go run ./cmd/librarymcp --db /path/to/library.db
```

Desktop HTTP server:

```text
http://127.0.0.1:8765/mcp
```

The HTTP server is started and stopped from Settings in the desktop app.

## Example MCP Client Config

```json
{
  "mcpServers": {
    "a3t-library": {
      "command": "go",
      "args": ["run", "./cmd/librarymcp"]
    }
  }
}
```

For installed usage, build a binary and point the client at it:

```sh
go build -o bin/librarymcp ./cmd/librarymcp
```

## Tools

- `search_books` - search book title, authors, and description.
- `search_passages` - search passages. Uses embeddings when LM Studio embeddings are configured and reachable, then falls back to FTS5.
- `get_book` - fetch one book by ID.
- `get_passage` - fetch one passage by ID.
- `list_books` - list recently imported books.

## Environment

- `LIBRARY_DB` selects the database path.
- `LIBRARY_LMSTUDIO_URL`, `LIBRARY_LMSTUDIO_EMBED_MODEL`, and `LIBRARY_LMSTUDIO_API_KEY` configure semantic passage search.
