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
- **Error handling** — always return errors, never panic in library code. Use `fmt.Errorf("context: %w", err)` for wrapping. When mount clients need behavior-specific handling, return/wrap typed sentinel errors instead of relying on string matching.

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

8. **Composite token routing** — sheet/bitable sub-entries (e.g. `shtcnXXX|sheetID`) use `TypeFile` for VFS display but `DriveAdapter.resolveCompositeType()` detects the token prefix to route Read/Write to the correct handler (`SheetHandler` / `BitableHandler`).

9. **WebDAV ContentTyper** — `vnodeFileInfo` implements `webdav.ContentTyper` to return MIME types based on file extension. This prevents `x/net/webdav`'s `findContentType` from opening and reading files during PROPFIND, which previously caused `Internal Server Error` appended to XML responses.

10. **WebDAV file creation** — `OpenFile` supports `os.O_CREATE` flag. `Operations.Create` creates docx documents via `docs +create` in the parent folder, then `io.Copy` writes body content via `docs +update`.

11. **FUSE errno mapping** — FUSE CRUD paths must not collapse all errors to `EIO`. VFS exposes `ErrReadOnly`, `ErrNotFound`, and `ErrUnsupported`; `pkg/mount/fuse.go` maps them to `EROFS`, `ENOENT`, and `ENOTSUP` so Finder, editors, and shell commands can react correctly. Unknown errors may still fall back to `EIO`.

12. **Control paths are not resource paths** — `_meta/`, `_ops/`, `_queries/`, and `_views/` are control-plane nodes. `Create`, `Mkdir`, `Remove`, and `Rename` must reject these paths unless the operation is explicitly handled by `writeControlNode` / query / op execution. Do not let inherited domain names trigger remote Drive CRUD against control nodes.

### lark-cli Response Format

All `lark-cli` API responses wrap data in a `data` field:
- **Raw API commands** (e.g. `drive files list`): `{"code": 0, "data": {...}}`
- **Skill commands** (prefixed with `+`, e.g. `sheets +info`): `{"ok": true, "data": {...}}`

Always unmarshal into `struct { Data struct { ... } \`json:"data"\` }`. Never expect top-level fields.

**Skill commands (`+`) vs raw commands:**
- Skill commands often do NOT support `--format json` (e.g. `sheets +info`, `base +table-list`). They always output JSON by default.
- Raw API commands (`drive files list`, `im chats list`) support `--format json`, `--page-all`, etc.
- When in doubt, run `lark-cli <cmd> --help` to check available flags.

## Testing

```bash
go test ./... -v -race -count=1
make test-cover
```

Tests exist for: `cmd/larkfs/test`, `pkg/cli/test`, `pkg/cache/test`, `pkg/naming` coverage via `pkg/cli/test`, `pkg/vfs/test`, `pkg/errors/test`, and selected adapter/mount behavior. Tests do NOT call real APIs — they test pure logic and mocked executor flows (caching, naming, retry, tree structure, control-node routing, error classification, JSON param safety, buffered writes).

All tests should live in each module's `test/` subdirectory (for example `pkg/vfs/test`) so test files do not sit flat beside implementation files. Keep tests black-box whenever possible; do not add exported production-only hooks just to reach unexported internals. Coverage for subdirectory tests must use `make test-cover`, which applies `-coverpkg` to production unit packages and excludes test driver packages from the denominator. `cmd/larkfs` and `pkg/mount` are exercised by black-box boundary tests but excluded from the unit coverage denominator because subprocess command execution and FUSE/WebDAV internals cannot be counted cleanly without same-package test files or test-only hooks.

When adding a new feature, add tests for the pure-logic parts. Adapter and mount tests should use mocked executors or VFS/mount fixtures, not real `lark-cli` network calls.

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
3. Existing commands: mount, unmount, serve, status, doctor, init, version

## Files to Watch

- `pkg/mount/common.go` — wiring center, all components assembled here
- `pkg/mount/fuse.go` — FUSE server, buffered file handles, CRUD errno mapping
- `pkg/mount/webdav.go` — WebDAV server, file creation, ContentTyper implementation
- `pkg/vfs/operations.go` — routing hub for all read/write/list/create operations
- `pkg/adapter/drive.go` — drive adapter with composite token routing (`resolveCompositeType`)
- `pkg/cli/executor.go` — all subprocess calls flow through here
- `pkg/doctype/types.go` — interface contract for all document handlers
- `cmd/larkfs/doctor.go` — system health checks, integrates `lark-cli doctor` JSON output
- `cmd/larkfs/init.go` — lark-cli setup wizard (config init + auth login)
