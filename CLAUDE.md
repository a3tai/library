# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with this repository.

## Project

A3T: Library is a Wails v3 desktop app (Go backend + Svelte 5 frontend) for indexing, browsing, reading, and full-text searching a local EPUB/PDF/text library. The packaged app/binary name is `Library`. The same SQLite index is exposed to AI clients through MCP, both as an embedded HTTP server inside the desktop app and as a standalone stdio binary.

## Commands

Development and build flows go through `wails3` / Taskfile. Frontend deps install lazily via Task.

- `task dev` - run the desktop app with hot reload (Vite on `WAILS_VITE_PORT`, default 9245).
- `task build` - production build for the current OS.
- `task package` - OS-dispatched package build. Outputs use `Library`, for example `bin/Library.app`.
- `wails3 task common:generate:bindings` - regenerate `frontend/bindings/` TS bindings from Go `Service` methods. Required after changing exported methods on `LibraryService`.
- `go run ./cmd/librarymcp [--db /path/to/library.db]` - stdio MCP server pointed at the same DB.
- `cd frontend && npm run check && npm run build && npm run test:e2e` - frontend type, build, and demo smoke checks.
- `go test ./... && go build ./... && go vet ./...` - backend test, compile, and vet checks.

## Architecture

### Process layout

- **`main.go`** wires the Wails `application.New(...)` with a single `LibraryService` bound to the frontend. The Svelte app calls Go via auto-generated bindings under `frontend/bindings/github.com/a3tai/library/`.
- **`libraryservice.go`** is the only Wails-bound service. Its public methods are the entire frontend API surface (`Snapshot`, `ListBooks`, `SearchPassages`, `GetBook`, `ImportPath`, `HydrateMetadata`, `StartMCPServer`, `StopMCPServer`, etc.). Adding a frontend-callable operation means adding a method here and re-running binding generation.
- **`cmd/librarymcp/main.go`** is a thin stdio entrypoint that opens the same `library.Store` and serves MCP over `stdin`/`stdout`.

### Internal packages

- **`internal/library`** - SQLite store. `Open(path)` runs migrations for books, passages, FTS tables, subjects/categories, embeddings, and settings. `DefaultDBPath()` resolves to `$LIBRARY_DB`, otherwise `<UserConfigDir>/A3T Library/library.db`; existing `<UserConfigDir>/Books/library.db` files are copied there once when the new database does not exist yet.
- **`internal/importer`** - walks a path, hashes each EPUB/PDF/text file (sha256 = book ID), parses metadata and text, chunks text into passages, and upserts through `Store`.
- **`internal/metadata`** - Open Library first (ISBN lookup, then title/author search), Google Books as fallback. Called from `LibraryService.HydrateMetadata`, which is auto-kicked after every import and gated by a busy mutex.
- **`internal/mcpserver`** - JSON-RPC 2.0 MCP protocol implementation over stdio and HTTP. Tools: `search_books`, `search_passages`, `get_book`, `get_passage`, `list_books`.

### Data model invariants

- A book's `id` is the sha256 of its file contents. Re-importing the same file is idempotent; moving a file updates `file_path` but keeps the same row.
- Passages have IDs of the form `<bookID>-<6-digit-index>` and are deleted/reinserted wholesale on every upsert.
- `UpsertImportedBook` and `UpdateMetadata` preserve populated existing fields when an incoming field is empty.
- `book_search` and `passage_search` FTS tables are kept in sync manually; any write path that touches `books` or `passages` content must also update the matching FTS row.

### Frontend

- Svelte 5 with runes (`$state`). `frontend/src/App.svelte` coordinates app state and routes into view components under `frontend/src/views`.
- All backend calls go through the generated `LibraryService` binding. Do not hand-edit anything under `frontend/bindings/`.

## Conventions Worth Knowing

- The DB pool uses WAL with `SetMaxOpenConns(4)` and an application-level writer mutex. Writes should respect `ctx` and avoid holding transactions longer than needed.
- `LibraryService.HydrateMetadata` is fire-and-forget from `ImportPath`; the `busy` flag lets the frontend show a hydrating indicator and prevents overlapping runs.
- Server-mode targets (`task build:server`, `task run:server`, `task build:docker`) exist in the Taskfile but rely on a `server` build tag that is not currently present in the Go sources. Treat them as scaffolding until that tag is implemented.
