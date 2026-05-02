import FileProvider
import Foundation
#if canImport(LarkFSNativeBridge)
import LarkFSNativeBridge
#endif

final class LarkFSFileProviderEnumerator: NSObject, NSFileProviderEnumerator {
    private let containerIdentifier: NSFileProviderItemIdentifier
    private let bridge: any LarkFSNativeBridge

    init(containerIdentifier: NSFileProviderItemIdentifier, bridge: any LarkFSNativeBridge) {
        self.containerIdentifier = containerIdentifier
        self.bridge = bridge
    }

    func invalidate() {}

    func enumerateItems(for observer: any NSFileProviderEnumerationObserver, startingAt page: NSFileProviderPage) {
        guard let path = pathForContainer() else {
            observer.finishEnumeratingWithError(LarkFSFileProviderError.noSuchItem(containerIdentifier))
            return
        }

        let observerBox = SendableBox(observer)
        Task {
            do {
                let children = try await bridge.children(of: path)
                let providerItems = children.map(LarkFSFileProviderItem.init(item:))
                let observer = observerBox.value
                observer.didEnumerate(providerItems)
                observer.finishEnumerating(upTo: nil)
            } catch {
                observerBox.value.finishEnumeratingWithError(LarkFSFileProviderError.bridge(error))
            }
        }
    }

    func enumerateChanges(
        for observer: any NSFileProviderChangeObserver,
        from syncAnchor: NSFileProviderSyncAnchor
    ) {
        observer.finishEnumeratingChanges(upTo: currentAnchor(), moreComing: false)
    }

    func currentSyncAnchor(completionHandler: @escaping (NSFileProviderSyncAnchor?) -> Void) {
        completionHandler(currentAnchor())
    }

    private func pathForContainer() -> String? {
        if containerIdentifier == .rootContainer || containerIdentifier == .workingSet {
            return "/"
        }
        return NativeBridgeIdentifier.path(for: containerIdentifier.rawValue)
    }

    private func currentAnchor() -> NSFileProviderSyncAnchor {
        NSFileProviderSyncAnchor(rawValue: Data("0".utf8))
    }
}

extension LarkFSFileProviderEnumerator: @unchecked Sendable {}
