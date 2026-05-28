# Development

A3T: Library is a Wails v3 app with a Go backend and a Svelte 5 frontend.

## Commands

Run the desktop app in development mode:

```sh
task dev
```

Build the app:

```sh
task build
```

Package the app for the current OS:

```sh
task package
```

Regenerate TypeScript bindings:

```sh
wails3 task common:generate:bindings
```

The binding task scans `./...` because exported `LibraryService` methods are split across feature files in the root package.

## Checks

```sh
./scripts/secret-scan.sh
gofmt -w .
cd frontend && npm ci && npm run check && npm run build
cd frontend && npm run test:e2e
go test ./...
go build ./...
```

`main.go` embeds `frontend/dist`, so clean checkouts need the frontend build before default Go package tests or builds.

The frontend smoke suite runs against `?demo` mode, so it does not require a native Wails runtime or a local library database.

## CI And Releases

- `.github/workflows/ci.yml` runs on pull requests and pushes to `main`.
- CI validates frontend types/build output, audits production npm dependencies, verifies `go.mod`/`go.sum`, checks generated Wails bindings, runs Go tests, and smoke-builds desktop artifacts on Linux, macOS, and Windows.
- `.github/workflows/release.yml` runs from tags like `v1.0.0`.
- Release tags must match `frontend/package.json`, `build/config.yml`, and `libraryservice.go`.
- Release artifacts are unsigned until platform signing credentials are configured; see `docs/RELEASE.md`.
- Public release artifacts get GitHub artifact attestations for provenance.

## Module Boundaries

The app follows the Go module guidance of keeping the desktop command at the repository root and implementation packages under `internal/`.

- Root `package main` owns Wails wiring, platform integration, and exported service methods.
- `internal/library` owns persistence and query behavior.
- `internal/importer`, `internal/metadata`, `internal/aimeta`, `internal/embedder`, and `internal/mcpserver` own feature-specific backend logic.
- Frontend code should stay feature-oriented: views in `frontend/src/views`, reusable controls in `frontend/src/components` or `frontend/src/lib/components`, and pure helpers in `frontend/src/lib`.

When a file crosses roughly 400-500 lines, split by behavior, not by arbitrary size. Prefer same-package file splits first when public APIs or Wails bindings would otherwise churn.

## Architecture

- `main.go` creates the Wails app, window, assets, and app menu.
- `libraryservice.go` is the frontend API boundary.
- `internal/library` owns SQLite storage, migrations, FTS5 search, settings, and embeddings metadata.
- `internal/importer` inventories files and extracts book metadata and text.
- `internal/metadata` enriches missing metadata from Open Library and Google Books.
- `internal/mcpserver` exposes the same store through MCP.
- `frontend/src/App.svelte` coordinates app state and view routing.

## Generated Files

Do not hand-edit `frontend/bindings/`. Regenerate it after exported Go service method changes.

`frontend/dist/`, `bin/`, app bundles, installers, Gradle build folders, and package artifacts are ignored by Git.

## Naming

The product name is `A3T: Library`. Package and binary artifacts use `Library`, so macOS packaging produces `Library.app`.

The Go module path is `github.com/a3tai/library`.

## Data Paths

New installs use:

```text
<UserConfigDir>/A3T Library/library.db
```

Existing pre-rename installs copy `<UserConfigDir>/Books/library.db` into the new `A3T Library` directory on first launch when the new database does not exist yet.

Use `LIBRARY_DB=/path/to/library.db` to point the app or MCP server at a specific database.
