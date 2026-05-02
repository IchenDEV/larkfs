import FileProvider
import Foundation
#if canImport(LarkFSNativeBridge)
import LarkFSNativeBridge
#endif

final class LarkFSFileProviderExtension: NSObject, NSFileProviderReplicatedExtension {
    private let domain: NSFileProviderDomain
    private let bridge: any LarkFSNativeBridge

    required init(domain: NSFileProviderDomain) {
        self.domain = domain
        self.bridge = LarkFSFileProviderBridgeFactory.makeBridge()
        super.init()
    }

    func invalidate() {}

    func item(
        for identifier: NSFileProviderItemIdentifier,
        request: NSFileProviderRequest,
        completionHandler: @escaping (NSFileProviderItem?, Error?) -> Void
    ) -> Progress {
        guard let path = path(for: identifier) else {
            completionHandler(nil, LarkFSFileProviderError.noSuchItem(identifier))
            return larkFSProgress(completed: 1)
        }

        let completionBox = SendableBox(completionHandler)
        Task {
            do {
                let item = try await bridge.item(at: path)
                completionBox.value(LarkFSFileProviderItem(item: item), nil)
            } catch {
                completionBox.value(nil, LarkFSFileProviderError.bridge(error))
            }
        }
        return larkFSProgress()
    }

    func fetchContents(
        for itemIdentifier: NSFileProviderItemIdentifier,
        version requestedVersion: NSFileProviderItemVersion?,
        request: NSFileProviderRequest,
        completionHandler: @escaping (URL?, NSFileProviderItem?, Error?) -> Void
    ) -> Progress {
        guard let path = path(for: itemIdentifier) else {
            completionHandler(nil, nil, LarkFSFileProviderError.noSuchItem(itemIdentifier))
            return larkFSProgress(completed: 1)
        }

        let outputURL = FileManager.default.temporaryDirectory
            .appendingPathComponent(UUID().uuidString)

        let completionBox = SendableBox(completionHandler)
        Task {
            do {
                let item = try await bridge.item(at: path)
                guard !item.isDirectory else {
                    completionBox.value(nil, nil, LarkFSFileProviderError.readOnly("Directories do not have downloadable contents."))
                    return
                }
                try await bridge.fetchContents(at: path, to: outputURL)
                completionBox.value(outputURL, LarkFSFileProviderItem(item: item), nil)
            } catch {
                completionBox.value(nil, nil, LarkFSFileProviderError.bridge(error))
            }
        }
        return larkFSProgress()
    }

    func createItem(
        basedOn itemTemplate: NSFileProviderItem,
        fields: NSFileProviderItemFields,
        contents url: URL?,
        options: NSFileProviderCreateItemOptions = [],
        request: NSFileProviderRequest,
        completionHandler: @escaping (NSFileProviderItem?, NSFileProviderItemFields, Bool, Error?) -> Void
    ) -> Progress {
        completionHandler(nil, fields, false, LarkFSFileProviderError.readOnly("File Provider writes are not enabled in this LarkFS build."))
        return larkFSProgress(completed: 1)
    }

    func modifyItem(
        _ item: NSFileProviderItem,
        baseVersion version: NSFileProviderItemVersion,
        changedFields: NSFileProviderItemFields,
        contents newContents: URL?,
        options: NSFileProviderModifyItemOptions = [],
        request: NSFileProviderRequest,
        completionHandler: @escaping (NSFileProviderItem?, NSFileProviderItemFields, Bool, Error?) -> Void
    ) -> Progress {
        completionHandler(nil, changedFields, false, LarkFSFileProviderError.readOnly("File Provider writes are not enabled in this LarkFS build."))
        return larkFSProgress(completed: 1)
    }

    func deleteItem(
        identifier: NSFileProviderItemIdentifier,
        baseVersion version: NSFileProviderItemVersion,
        options: NSFileProviderDeleteItemOptions = [],
        request: NSFileProviderRequest,
        completionHandler: @escaping (Error?) -> Void
    ) -> Progress {
        completionHandler(LarkFSFileProviderError.readOnly("File Provider deletes are not enabled in this LarkFS build."))
        return larkFSProgress(completed: 1)
    }

    func enumerator(
        for containerItemIdentifier: NSFileProviderItemIdentifier,
        request: NSFileProviderRequest
    ) throws -> any NSFileProviderEnumerator {
        LarkFSFileProviderEnumerator(containerIdentifier: containerItemIdentifier, bridge: bridge)
    }

    private func path(for identifier: NSFileProviderItemIdentifier) -> String? {
        if identifier == .rootContainer || identifier == .workingSet {
            return "/"
        }
        return NativeBridgeIdentifier.path(for: identifier.rawValue)
    }
}

extension LarkFSFileProviderExtension: @unchecked Sendable {}

enum LarkFSFileProviderBridgeFactory {
    static func makeBridge() -> any LarkFSNativeBridge {
        if let override = ProcessInfo.processInfo.environment["LARKFS_BIN"], !override.isEmpty {
            return LarkFSCLIFileProviderBridge(binaryURL: URL(fileURLWithPath: override))
        }

        if let bundled = Bundle.main.resourceURL?
            .appendingPathComponent("bin", isDirectory: true)
            .appendingPathComponent("larkfs"),
           FileManager.default.isExecutableFile(atPath: bundled.path) {
            return LarkFSCLIFileProviderBridge(binaryURL: bundled)
        }

        let path = ProcessInfo.processInfo.environment["PATH"] ?? ""
        for entry in path.split(separator: ":").map(String.init) {
            let candidate = URL(fileURLWithPath: entry).appendingPathComponent("larkfs")
            if FileManager.default.isExecutableFile(atPath: candidate.path) {
                return LarkFSCLIFileProviderBridge(binaryURL: candidate)
            }
        }

        return LarkFSCLIFileProviderBridge(binaryURL: URL(fileURLWithPath: "/usr/local/bin/larkfs"))
    }
}
