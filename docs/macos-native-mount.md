# macOS 原生挂载方案

目标：把 LarkFS 做成 macOS 能原生管理的文件提供方，体验尽量接近 OneDrive。FUSE / WebDAV 继续保留为兼容路径。

## 为什么走 File Provider

- Finder 侧边栏、按需下载、离线缓存、系统级同步语义，都是 `File Provider` 的职责范围。
- FUSE 适合先把远端资源变成本地树；macOS 云盘集成的主路径应走 File Provider。
- WebDAV 适合作为兼容后端，不适合作为长期的原生体验方案。

## 这次完成的第一阶段

- 新增 `apps/LarkFSDesktop` SwiftUI 宿主 app。
- 它直接读取 `larkfs version --json`、`larkfs doctor --json`、`larkfs status --json`，不再解析面向人的纯文本输出。
- root 级 `script/build_and_run.sh` 会优先走 Xcode project，产出带 `LarkFSFileProvider.appex` 的 `.app`；没有 project 时回退到 SwiftPM 预览包。
- app 里已经显式建模了 File Provider 目标域和原生挂载 readiness，并提供注册 / 移除 domain 的入口。

## File Provider 起步版本

这一步已经把原生挂载的代码边界先放进仓库：

- `larkfs native item <path>`：返回单个路径的 File Provider 元数据。
- `larkfs native list <path>`：返回目录子项。
- `larkfs native fetch <path> --output <file>`：把文件内容写到系统临时文件，供 File Provider materialize。
- `apps/LarkFSDesktop/NativeBridge`：Swift 侧共享 bridge，负责调用 `larkfs native`。
- `apps/LarkFSDesktop/FileProviderExtension`：可编译的 `NSFileProviderReplicatedExtension` 骨架，第一版只读，覆盖枚举、元数据查询和内容下载。
- `apps/LarkFSDesktop/FileProviderSupport`：给 Xcode target 使用的 Info.plist / entitlement 模板，包含 document group、App Group、sandbox 和网络访问。

这版先保持只读。写回、重命名、删除要等只读枚举和下载在 Finder 中跑通后再接，因为 File Provider 的本地变更回调会涉及冲突处理和版本语义。

## 建议的完整架构

1. `LarkFSDesktop.app`
   - 负责登录状态、挂载状态、用户设置、诊断、域管理。
   - 提供原生挂载的控制面。

2. `File Provider Extension`
   - 真正把飞书资源暴露给 Finder。
   - 负责枚举、物化、占位文件、按需下载、写回。

3. `larkfs` 本地 bridge
   - 继续复用现有 Go 侧的 adapter / vfs / cache 能力。
   - extension 通过本地 bridge 或共享缓存层复用现有飞书协议能力。

## 下一步怎么接

1. 用 Apple Development / Developer ID 证书构建：

   ```bash
   LARKFS_DEVELOPMENT_TEAM=<TEAM_ID> ./script/build_and_run.sh
   ```

2. 打开 app，在 Native Mount 面板里点 Register，把 `dev.ichen.larkfs` domain 加进系统。
3. 在 Finder 里先验证只读枚举和按需下载。
4. 再补写回、重命名、删除、冲突处理和 working set 增量同步。

## 当前限制

- 当前仓库已经能通过 Xcode 构建带 `.appex` 的 app，并能通过 `./script/build_and_run.sh --verify` 启动验证。
- 当前机器没有可用代码签名身份，`security find-identity -p codesigning -v` 返回 `0 valid identities found`。ad-hoc 签名可以嵌入 entitlements，但 `pluginkit` 仍不会列出这个 File Provider 插件。
- 只读 bridge 已经成型；Finder 侧真实注册需要有效签名后继续验证，同步回调和写回还没做。
