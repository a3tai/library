# Contributing

Thanks for helping improve A3T: Library.

## Development Setup

Install Go, Node.js/npm, Wails v3, and Task. Then run:

```sh
task dev
```

Frontend dependencies are installed by the Taskfile as needed.

## Before Opening A Pull Request

Run:

```sh
./scripts/secret-scan.sh
gofmt -w .
go test ./...
go build ./...
go vet ./...
cd frontend && npm run check && npm run build && npm run test:e2e
```

If you change exported methods on `LibraryService`, regenerate bindings:

```sh
wails3 task common:generate:bindings
```

Do not hand-edit files under `frontend/bindings/`.

## Pull Request Scope

Keep changes focused. Include a short explanation of behavior changes, migration impact, and manual verification. For packaging changes, mention the OS and artifact you tested.

## License

By contributing, you agree that your contribution is licensed under the MIT license used by this repository.
