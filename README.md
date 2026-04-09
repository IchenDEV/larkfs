<p align="center">
  <h1 align="center">LarkFS</h1>
  <p align="center">Mount Lark/Feishu as a local filesystem — FUSE & WebDAV</p>
</p>

<p align="center">
  <a href="https://github.com/IchenDEV/larkfs/actions"><img src="https://github.com/IchenDEV/larkfs/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/IchenDEV/larkfs/releases"><img src="https://img.shields.io/github/v/release/IchenDEV/larkfs?label=release" alt="Release"></a>
  <a href="https://github.com/IchenDEV/larkfs/blob/main/LICENSE"><img src="https://img.shields.io/github/license/IchenDEV/larkfs" alt="License"></a>
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux-blue" alt="Platform">
</p>

---

将飞书/Lark 的 7 大业务域（Drive、Wiki、IM、Calendar、Tasks、Mail、Meetings）映射为本地可读写的文件目录。像操作本地文件一样操作云端资源。

## Why

- `cat /mnt/lark/wiki/产品文档/PRD.md` — 直接用终端 / 编辑器读写飞书文档
- `ls /mnt/lark/im/团队群/` — 浏览消息、发送文本
- `echo "明天 10:00 周会" > /mnt/lark/calendar/_create.md` — 创建日历事件
- 在 Finder / Nautilus 中拖拽文件到飞书云空间

## Features

| 能力 | 说明 |
|---|---|
| **FUSE 挂载** | 完整 POSIX 语义，像本地目录一样 `ls / cat / vim / cp` |
| **WebDAV 模式** | 无需内核模块，Finder / 文件管理器直连 |
| **多类型文档** | docx → Markdown, sheet → 目录/CSV, bitable → 目录/JSONL, file → 原样下载 |
| **7 大域** | Drive · Wiki · IM · Calendar · Tasks · Mail · Meetings |
| **同名冲突解决** | 自动 `name~token` 后缀，持久化映射保证路径稳定 |
| **三级缓存** | in-memory → 磁盘 LRU ContentCache → 远程拉取，TTL 自动过期 |
| **重试 & 认证恢复** | API 限流自动指数退避，token 过期自动刷新 |
| **守护进程** | 后台运行、PID 管理、优雅关闭、stale mount 自动清理 |
| **跨平台** | macOS (macFUSE / Fuse-T) + Linux (FUSE3) |
| **CI/CD** | GitHub Actions + GoReleaser 自动发布多平台二进制 |

## Quick Start

### Prerequisites

- **[lark-cli](https://github.com/larksuite/cli)** — `npm install -g @larksuite/cli`
- **FUSE** (仅 FUSE 模式需要):
  - macOS: `brew install macfuse` 或 `brew install macos-fuse-t/homebrew-cask/fuse-t`
  - Linux: `apt install fuse3`

### Install

```bash
# Go install
go install github.com/IchenDEV/larkfs/cmd/larkfs@latest

# 或从 Releases 下载预编译二进制
# https://github.com/IchenDEV/larkfs/releases
```

### Setup

```bash
# 一键初始化：检测 lark-cli → 配置应用 → OAuth 登录
larkfs init
```

`larkfs init` 会自动引导完成以下步骤：
1. 检查 lark-cli 是否安装
2. 若未配置，运行 `lark-cli config init --new` 创建应用
3. 若未登录，运行 `lark-cli auth login --domain all` 获取授权

### Usage

```bash
# 检查环境（lark-cli 配置、认证、连通性、FUSE）
larkfs doctor

# WebDAV 模式（推荐，无需 FUSE）
larkfs serve --port 8080

# 挂载（前台）
larkfs mount ~/lark

# 挂载（后台守护进程）
larkfs mount ~/lark -d

# 只读模式
larkfs mount ~/lark --read-only

# 指定域
larkfs mount ~/lark --domains drive,wiki,calendar

# 查看状态
larkfs status

# 卸载
larkfs unmount ~/lark

# 卸载全部
larkfs unmount --all
```

## Directory Layout

```
~/lark/
├── drive/                     # 云空间
│   ├── 项目文档.md            # docx → Markdown (读写)
│   ├── 数据表.sheet/          # spreadsheet → 目录
│   │   ├── _meta.json
│   │   └── Sheet1.csv         # 每个 sheet → CSV (读写)
│   ├── 多维表格.base/         # bitable → 目录
│   │   ├── _meta.json
│   │   └── 表1.jsonl          # 每张表 → JSONL (读写)
│   └── 设计稿.sketch          # 普通文件 → 原样下载
├── wiki/                      # 知识库
│   └── 产品空间/
│       ├── PRD.md             # wiki node → docx → Markdown
│       └── 数据看板.sheet/
├── im/                        # 即时消息
│   └── 产品群/
│       ├── latest.md          # 最新消息 (只读)
│       ├── _send.md           # 写入即发送
│       └── files/             # 群文件
├── calendar/                  # 日历
│   ├── 周一站会.md            # 事件详情 (只读)
│   └── _create.md             # 写入即创建事件
├── tasks/                     # 任务
│   ├── 完成设计评审.md        # 任务详情
│   └── _create.md             # 写入即创建任务
├── mail/                      # 邮箱
│   ├── INBOX/
│   │   └── 2026-04-07_张三_会议通知.md
│   ├── _compose.md            # 写入即发送
│   └── _send.md
└── meetings/                  # 会议
    └── 2026-04-07/
        └── 产品评审/
            ├── _meta.json     # 会议元数据
            ├── summary.md     # AI 摘要
            ├── todos.md       # 待办提取
            ├── transcript.md  # 逐字稿
            └── recording.mp4  # 录制文件
```

## Architecture

```
┌─────────────┐   ┌─────────────┐
│  FUSE mount │   │ WebDAV srv  │    ← mount layer (pkg/mount)
└──────┬──────┘   └──────┬──────┘
       │                 │
       └────────┬────────┘
                │
         ┌──────┴──────┐
         │ VFS + Tree  │              ← virtual fs (pkg/vfs)
         └──────┬──────┘
                │
    ┌───────────┼───────────┐
    │     Domain Adapters   │         ← adapters (pkg/adapter)
    │  drive wiki im cal    │
    │  task  mail meeting   │
    └───────────┬───────────┘
                │
         ┌──────┴──────┐
         │  DocType     │             ← type handlers (pkg/doctype)
         │  Registry    │                docx/sheet/bitable/file/folder
         └──────┬──────┘
                │
         ┌──────┴──────┐
         │  CLI Exec   │             ← lark-cli wrapper (pkg/cli)
         │  + Retry    │                with middleware, retry, auth
         └─────────────┘
```

**Key packages:**

| Package | Responsibility |
|---|---|
| `cmd/larkfs` | CLI entry — mount, unmount, serve, status, doctor, init |
| `pkg/cli` | lark-cli subprocess wrapper, JSON param builder, error classification |
| `pkg/doctype` | Per-type read/write handlers: docx, sheet, bitable, file, folder, readonly |
| `pkg/adapter` | Domain adapters: drive, wiki, im, calendar, task, mail, meeting |
| `pkg/vfs` | Virtual tree + operations routing |
| `pkg/mount` | FUSE server (go-fuse/v2), WebDAV server (x/net/webdav) |
| `pkg/cache` | Metadata TTL cache + LRU disk content cache |
| `pkg/naming` | Name conflict resolution with `~token` suffix + persistent mapping |
| `pkg/daemon` | PID file, fork, health check, stale mount cleanup |
| `pkg/errors` | Retry with exponential backoff, auth recovery, errno mapping |
| `pkg/config` | Mount/Serve config structs, path resolution |

## Development

```bash
# Build
make build

# Run tests
make test

# Lint
make lint

# Dev mount (foreground, debug log)
make dev-mount

# Dev unmount
make dev-unmount

# Clean
make clean
```

### Release

Releases are automated via GitHub Actions + GoReleaser on version tags:

```bash
git tag v0.1.0
git push origin v0.1.0
```

Produces multi-platform binaries (linux/darwin × amd64/arm64).

## Configuration

All runtime state is stored in `~/.larkfs/`:

```
~/.larkfs/
├── cache/          # LRU content cache (default 500MB)
├── mounts/         # PID files for active mounts
├── namemap.json    # Persistent name → token mappings
└── larkfs.log      # Log file
```

### CLI Flags

| Flag | Default | Description |
|---|---|---|
| `--daemon, -d` | `false` | Run as background daemon |
| `--cache-dir` | `~/.larkfs/cache` | Cache directory |
| `--metadata-ttl` | `60` | Metadata cache TTL (seconds) |
| `--read-only` | `false` | Mount in read-only mode |
| `--domains` | all 7 | Comma-separated enabled domains |
| `--lark-cli` | auto-detect | Path to lark-cli binary |
| `--log-level` | `info` | Log level (debug/info/warn/error) |

## License

[MIT](LICENSE)
