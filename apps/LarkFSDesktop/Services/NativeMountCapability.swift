import Foundation
#if canImport(FileProvider)
import FileProvider
#endif

struct NativeMountCapability {
    let fileProviderSupported: Bool
    let domainIdentifier: String
    let bridgeBinaryFound: Bool
    let planDocumentFound: Bool
    let extensionPackaged: Bool
    let workspaceRoot: URL?
    let binaryURL: URL?
    let notes: [String]

    static func current() -> NativeMountCapability {
        let binaryURL = BundlePaths.larkfsBinaryURL
        let extensionURL = Bundle.main.builtInPlugInsURL?
            .appendingPathComponent("LarkFSFileProvider.appex")
        return NativeMountCapability(
            fileProviderSupported: fileProviderRuntimeAvailable,
            domainIdentifier: fileProviderDomainIdentifier,
            bridgeBinaryFound: binaryURL != nil,
            planDocumentFound: BundlePaths.nativeMountPlanURL != nil,
            extensionPackaged: extensionURL.map { FileManager.default.fileExists(atPath: $0.path) } ?? false,
            workspaceRoot: BundlePaths.workspaceRoot,
            binaryURL: binaryURL,
            notes: [
                "宿主 app 已经能读取 larkfs 的结构化状态，并能作为 File Provider 的控制面。",
                "仓库里已经有 SwiftPM 可编译的 File Provider 扩展模块和 larkfs native bridge。",
                "真正出现在 Finder 侧边栏，还需要用完整 Xcode 增加 app extension target、entitlements、签名和系统注册。",
            ]
        )
    }
}

#if canImport(FileProvider)
private let fileProviderDomainIdentifier = NSFileProviderDomainIdentifier("dev.ichen.larkfs").rawValue
private let fileProviderRuntimeAvailable = true
#else
private let fileProviderDomainIdentifier = "dev.ichen.larkfs"
private let fileProviderRuntimeAvailable = false
#endif
