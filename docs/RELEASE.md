# Release Checklist

Use this checklist before promoting a tag as a public end-user release.

## Version Gate

Release tags must use `vX.Y.Z` and match:

- `frontend/package.json`
- `build/config.yml`
- `libraryservice.go`

The release workflow enforces this before it builds artifacts.

## Required Checks

Run the same checks as CI before tagging:

```sh
./scripts/secret-scan.sh
go test ./...
go build ./...
go vet ./...
cd frontend && npm ci && npm run check && npm run build && npm run test:e2e
wails3 generate bindings -clean=true -ts ./...
git diff --exit-code -- go.mod go.sum frontend/bindings
```

## Signing Status

The public GitHub release workflow currently builds unsigned artifacts. Do not
present those artifacts as production-trusted installers until platform signing
credentials are configured.

Supported local signing tasks already exist:

- macOS: `wails3 task darwin:sign:notarize`
- Windows executable: `wails3 task windows:sign`
- Windows installer: `wails3 task windows:sign:installer`
- Linux packages: `wails3 task linux:sign:packages`

Keep signing certificates, app-specific passwords, keychain profiles, PGP keys,
and notarization credentials outside the repository. Store CI credentials only as
repository or environment secrets.

## Public History

Before pushing the first public `main`, verify history does not expose private
data or bulky generated artifacts:

```sh
git rev-list --all --objects | grep -E 'frontend/screenshots|frontend/public/(Inter-Medium.ttf|svelte.svg|wails.png)|build/appicon.psd' || true
git log --all --format='%h %an <%ae>'
```

If private history has already been pushed, replace the public branch with a new
clean root commit from the sanitized tree and rotate any exposed credentials.
