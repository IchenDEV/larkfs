import Foundation

enum BundlePaths {
    static var workspaceRoot: URL? {
        if let explicitRoot = Bundle.main.object(forInfoDictionaryKey: "LarkFSWorkspaceRoot") as? String {
            return URL(fileURLWithPath: explicitRoot, isDirectory: true)
        }

        let candidate = Bundle.main.bundleURL
            .deletingLastPathComponent()
            .deletingLastPathComponent()

        if FileManager.default.fileExists(atPath: candidate.appendingPathComponent("go.mod").path) {
            return candidate
        }
        return nil
    }

    static var larkfsBinaryURL: URL? {
        if let override = ProcessInfo.processInfo.environment["LARKFS_BIN"], !override.isEmpty {
            let url = URL(fileURLWithPath: override)
            if FileManager.default.isExecutableFile(atPath: url.path) {
                return url
            }
        }

        if let bundledResource = Bundle.main.resourceURL?
            .appendingPathComponent("bin", isDirectory: true)
            .appendingPathComponent("larkfs"),
           FileManager.default.isExecutableFile(atPath: bundledResource.path) {
            return bundledResource
        }

        if let workspaceRoot {
            let repoBinary = workspaceRoot.appendingPathComponent("bin/larkfs")
            if FileManager.default.isExecutableFile(atPath: repoBinary.path) {
                return repoBinary
            }
        }

        let paths = (ProcessInfo.processInfo.environment["PATH"] ?? "")
            .split(separator: ":")
            .map(String.init)
        for path in paths {
            let candidate = URL(fileURLWithPath: path).appendingPathComponent("larkfs")
            if FileManager.default.isExecutableFile(atPath: candidate.path) {
                return candidate
            }
        }
        return nil
    }

    static var nativeMountPlanURL: URL? {
        guard let workspaceRoot else { return nil }
        let fileURL = workspaceRoot.appendingPathComponent("docs/macos-native-mount.md")
        guard FileManager.default.fileExists(atPath: fileURL.path) else { return nil }
        return fileURL
    }

    static let configDirectory = URL(fileURLWithPath: NSHomeDirectory(), isDirectory: true)
        .appendingPathComponent(".larkfs", isDirectory: true)
}
