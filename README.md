# A3T: Library

[![CI](https://github.com/a3tai/library/actions/workflows/ci.yml/badge.svg)](https://github.com/a3tai/library/actions/workflows/ci.yml)

A3T: Library is a local-first desktop app for indexing, browsing, reading, and searching a personal book library. It imports EPUB, PDF, and text files into a local SQLite index, then exposes that same index through the app UI and an optional MCP server for AI clients.

The packaged desktop app is named `Library` (`Library.app`, `Library.exe`, or `bin/Library`). The user-facing product name is `A3T: Library`.

## Features

- Import folders of EPUB, PDF, and text files.
- Browse by library, recent imports, authors, subjects, categories, and unprocessed books.
- Search books and passages with SQLite FTS5.
- Read extracted passages in the built-in reader.
- Refresh missing metadata from Open Library first, then Google Books.
- Optionally use LM Studio for AI metadata, semantic passage search, and chat.
- Expose the local index through MCP over stdio or the app's local HTTP server.

## Requirements

- Go 1.25 or newer.
- Node.js and npm.
- Wails v3 CLI (`wails3`).
- Task (`task`) for the checked-in build workflows.

Linux builds also need the WebKitGTK and GTK development packages expected by Wails.

## Development

```sh
task dev
```

Useful checks:

```sh
cd frontend && npm ci && npm run check && npm run build
cd frontend && npm run test:e2e
go test ./...
go build ./...
```

Regenerate Wails TypeScript bindings after changing exported `LibraryService` methods:

```sh
wails3 task common:generate:bindings
```

Generated bindings live under `frontend/bindings/` and should not be edited by hand.

## Build And Package

Build the desktop app:

```sh
task build
```

Package for the current OS:

```sh
task package
```

Expected output names use `Library`, for example `bin/Library.app` on macOS and `bin/Library.exe` on Windows. If app metadata in `build/config.yml` changes, refresh platform assets with:

```sh
wails3 task common:update:build-assets
```

## CI And Releases

Pull requests and pushes to `main` run GitHub Actions validation and desktop build smoke tests on Linux, macOS, and Windows.

Create a release by pushing a version tag that matches the app metadata:

```sh
git tag v1.0.0
git push origin v1.0.0
```

The release workflow publishes unsigned desktop artifacts and `SHA256SUMS.txt` to the GitHub Release. Signing and notarization are intentionally separate from the public workflow until distribution certificates are configured. See [docs/RELEASE.md](docs/RELEASE.md) before promoting a tag as a public end-user release.

## MCP Server

Run the standalone stdio MCP server against the default app database:

```sh
go run ./cmd/librarymcp
```

Use a specific database:

```sh
go run ./cmd/librarymcp --db /path/to/library.db
```

The desktop app can also start a local HTTP MCP server from Settings. By default it binds to `127.0.0.1:8765` at `/mcp`.

See [docs/MCP.md](docs/MCP.md) for MCP tool details.

## Data And Configuration

By default, new installs store data at:

```text
<UserConfigDir>/A3T Library/library.db
```

Existing pre-rename installs copy `<UserConfigDir>/Books/library.db` into the new `A3T Library` directory on first launch when the new database does not exist yet.

Environment variables:

- `LIBRARY_DB` - override the SQLite database path.
- `LIBRARY_LMSTUDIO_URL` - OpenAI-compatible LM Studio base URL, ending in `/v1`.
- `LIBRARY_LMSTUDIO_MODEL` - chat and metadata extraction model.
- `LIBRARY_LMSTUDIO_EMBED_MODEL` - embedding model for semantic passage search.
- `LIBRARY_LMSTUDIO_API_KEY` - optional bearer token for LM Studio.
- `LIBRARY_PICKER=native` - try the native Wails file picker path on macOS.
- `LIBRARY_OTLP_ENDPOINT` and `LIBRARY_TRACE_STDOUT` - tracing controls for development.

See `.env.example` for a copyable local-development template. Real `.env` files are ignored and should not be committed.

## Privacy

Library data, extracted text, settings, and embeddings are stored locally. Metadata enrichment calls Open Library and Google Books. AI features call the configured OpenAI-compatible endpoint, intended for a local LM Studio server by default.

## Repository Layout

- `main.go` - Wails desktop app entrypoint and menu setup.
- `libraryservice.go` - Wails-bound backend API used by the frontend.
- `internal/library` - SQLite schema, migrations, FTS, settings, and query code.
- `internal/importer` - EPUB, PDF, and text inventory/import pipeline.
- `internal/metadata` - Open Library and Google Books metadata enrichment.
- `internal/mcpserver` - JSON-RPC MCP implementation over stdio and HTTP.
- `frontend/src` - Svelte 5 app and UI components.
- `build/` - Wails platform build and packaging assets.

## Community

See [CONTRIBUTING.md](CONTRIBUTING.md).

Security vulnerabilities should be reported privately through [SECURITY.md](SECURITY.md). Support expectations are in [SUPPORT.md](SUPPORT.md). Project conduct expectations are in [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

## License

MIT. See [LICENSE](LICENSE).
