# LarkFS — Agent Guidelines

## Project Overview

LarkFS is a virtual filesystem that mounts Lark/Feishu cloud resources as local directories via FUSE or WebDAV. It wraps `lark-cli` (the official Lark CLI tool) for all API interactions.

## Tech Stack

- **Language**: Go 1.25+
- **CLI**: `github.com/spf13/cobra`
- **FUSE**: `github.com/hanwen/go-fuse/v2`
- **WebDAV**: `golang.org/x/net/webdav`
- **Logging**: `log/slog` (stdlib structured logging)
- **CI**: GitHub Actions + `golangci-lint` + GoReleaser

## Code Style

- **Functional over OOP** — prefer standalone functions, use structs only when state is needed.
- **File size** — keep files under 300 lines (ideal ~100). If a file grows beyond that, split it.
- **No class abuse** — no inheritance, no deep type hierarchies.
- **Comments** — only for non-obvious intent. Never narrate what the code does.
- **Error handling** — always return errors, never panic in library code. Use `fmt.Errorf("context: %w", err)` for wrapping.

## Architecture Rules

### Dependency Direction (strict, no cycles)

```
cmd/larkfs → pkg/mount → pkg/vfs → pkg/adapter → pkg/doctype → pkg/cli
                       → pkg/cache
                       → pkg/errors (uses pkg/cli error types)
                       → pkg/naming
                       → pkg/daemon → pkg/config
```

### Key Design Decisions

1. **`pkg/cli` has no external pkg imports** — it defines error types (`ErrAuthExpired`, `ErrNotFound`, etc.) and the `Executor`. Other packages import cli, never the reverse.

2. **Retry/Auth is injected via middleware** — `Executor.SetMiddleware()` accepts a callback. `pkg/mount/common.go` wires `pkg/errors.WithRetry` + `AuthRecovery` into the executor without creating circular deps.

3. **`TypeHandler` interface** — all document types (docx, sheet, bitable, file, folder, readonly) implement the same interface with `ctx context.Context` on every method. New doc types = new handler + register in `pkg/doctype/registry.go`.

4. **JSON params** — always use `cli.JSONParam(map[string]any{...})` (backed by `json.Marshal`). Never concatenate strings into JSON. This prevents injection.

5. **Name conflicts** — `naming.Resolver` uses `name~shortToken` suffix. Mappings are persisted in `~/.larkfs/namemap.json`. When conflicts arise, ALL conflicting entries get suffixed (not just the new one).

6. **VFS tree TTL** — `VNode.NeedsRefresh(ttl)` controls when `ReadDir` re-fetches from API vs returns cached children. TTL is configurable via `--metadata-ttl`.

7. **Domain filtering** — `vfs.NewTree(domains []string)` only creates nodes for enabled domains. Controlled by `--domains` flag.

## Testing

```bash
go test ./... -v -race -count=1
```

Tests exist for: `pkg/cli`, `pkg/cache`, `pkg/naming`, `pkg/vfs`, `pkg/errors`. Tests do NOT call real APIs — they test pure logic (caching, naming, retry, tree structure, error classification, JSON param safety).

When adding a new feature, add tests for the pure-logic parts. Adapter and mount tests require mocking the executor (not yet implemented).

## Common Tasks

### Add a new document type

1. Create `pkg/doctype/newtype.go` implementing `TypeHandler`
2. Register in `pkg/doctype/registry.go` → `NewRegistry()`
3. Add to `DocType` constants in `pkg/doctype/types.go`
4. Update `IsReadOnly()`, `IsDirectory()`, `FileExtension()` if needed

### Add a new domain

1. Create `pkg/adapter/newdomain.go` with adapter struct
2. Wire into `pkg/mount/common.go` → `buildMount()`
3. Add field to `vfs.Operations` + `OperationsConfig`
4. Handle in `fetchEntries()`, `readContent()`, `writeContent()` switch cases
5. Add domain name to the default `--domains` value in `cmd/larkfs/mount.go`

### Add a new CLI command

1. Create `cmd/larkfs/newcmd.go` with `newXxxCmd() *cobra.Command`
2. Register in `cmd/larkfs/main.go` → `root.AddCommand()`

## Files to Watch

- `pkg/mount/common.go` — wiring center, all components assembled here
- `pkg/vfs/operations.go` — routing hub for all read/write/list operations
- `pkg/cli/executor.go` — all subprocess calls flow through here
- `pkg/doctype/types.go` — interface contract for all document handlers

## Cursor Cloud specific instructions

### System dependencies (pre-installed in snapshot)

- **Go 1.25+** — already available via `gotip`/`go`
- **fuse3 + libfuse3-dev** — compile-time dependency for `go-fuse/v2`
- **lark-cli** — installed globally via `npm install -g @larksuite/cli`; required at runtime by all CLI/mount/serve commands
- **golangci-lint** — installed at `$(go env GOPATH)/bin/golangci-lint`

### PATH note

`$(go env GOPATH)/bin` is added to PATH in `~/.bashrc`. If `golangci-lint` is not found, run `export PATH="$PATH:$(go env GOPATH)/bin"`.

### Quick reference

| Task | Command |
|------|---------|
| Build | `make build` |
| Test | `make test` |
| Lint | `make lint` |
| Run WebDAV (dev) | `./bin/larkfs serve --port 8080 --log-level debug` |
| Run FUSE mount (dev) | `make dev-mount` |
| Check env | `./bin/larkfs doctor` |

### Caveats

- `larkfs serve` and `larkfs mount` both require `lark-cli` to be in PATH **and** authenticated (`lark-cli auth login`). Without auth, the server starts but API calls fail with auth errors. Unit tests (`make test`) do **not** require `lark-cli` or auth.
- `make lint` has a pre-existing `errcheck` warning in `pkg/cache/content_test.go:73` — this is not a regression.
- FUSE mount mode (`larkfs mount`) requires the FUSE kernel module loaded. In containerized environments this may not work; use WebDAV mode (`larkfs serve`) instead.
