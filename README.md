# LarkFS - 飞书虚拟文件系统

将飞书/Lark 的 Drive、Wiki、IM、Calendar、Tasks、Mail、Meetings 映射为本地可读写的文件目录。

## 特性

- **FUSE 挂载** — 像操作本地文件一样读写飞书资源
- **WebDAV 模式** — 无需 FUSE 内核模块，Finder/文件管理器直连
- **多类型文档** — docx→Markdown, sheet→CSV, bitable→JSONL, file→原样
- **7 大域** — Drive、Wiki、IM、Calendar、Tasks、Mail、Meetings
- **同名冲突解决** — 自动 `name~token` 后缀避免文件名碰撞
- **守护进程** — 后台运行、优雅关闭、健康检查
- **跨平台** — macOS (macFUSE/Fuse-T) + Linux (FUSE3)

## 前置依赖

- [lark-cli](https://github.com/larksuite/cli) — 已安装并完成 `lark-cli auth login`
- macOS: `brew install macos-fuse-t/homebrew-cask/fuse-t` 或 macFUSE
- Linux: `apt install fuse3`

## 安装

```bash
go install github.com/IchenDEV/larkfs/cmd/larkfs@latest
```

或从 [Releases](https://github.com/IchenDEV/larkfs/releases) 下载预编译二进制。

## 使用

```bash
# 检查环境
larkfs doctor

# FUSE 挂载（前台）
larkfs mount /mnt/lark

# FUSE 挂载（后台守护）
larkfs mount /mnt/lark -d

# WebDAV 模式
larkfs serve --port 8080

# 查看挂载状态
larkfs status

# 卸载
larkfs unmount /mnt/lark
```

## 目录结构

```
/mnt/lark/
├── drive/         # 云空间 (docx→.md, sheet→.sheet/, bitable→.base/, file→原样)
├── wiki/          # 知识库 (异构: docx/sheet/bitable 混合)
├── im/            # 消息 (latest.md 查看, _send.md 发送)
├── calendar/      # 日历 (事件 .md, _create.md 创建)
├── tasks/         # 任务 (任务 .md, _create.md 创建)
├── mail/          # 邮箱 (INBOX/SENT/DRAFT, frontmatter+body)
└── meetings/      # 会议 (summary.md, transcript.md, recording.mp4)
```

## 开发

```bash
make build       # 编译
make test        # 测试
make lint        # 代码检查
make dev-mount   # 开发模式挂载到 /tmp/larkfs
make dev-unmount # 卸载
```

## License

MIT