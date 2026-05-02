import FileProvider
import Foundation
#if canImport(LarkFSNativeBridge)
import LarkFSNativeBridge
#endif
import UniformTypeIdentifiers

final class LarkFSFileProviderItem: NSObject, NSFileProviderItem {
    private let item: NativeBridgeItem

    init(item: NativeBridgeItem) {
        self.item = item
    }

    var itemIdentifier: NSFileProviderItemIdentifier {
        if item.id == NativeBridgeIdentifier.root {
            return .rootContainer
        }
        return NSFileProviderItemIdentifier(item.id)
    }

    var parentItemIdentifier: NSFileProviderItemIdentifier {
        guard let parentID = item.parentID, !parentID.isEmpty else {
            return .rootContainer
        }
        if parentID == NativeBridgeIdentifier.root {
            return .rootContainer
        }
        return NSFileProviderItemIdentifier(parentID)
    }

    var filename: String {
        item.name
    }

    var contentType: UTType {
        if item.isDirectory {
            return .folder
        }
        return UTType(item.contentType) ?? .data
    }

    var capabilities: NSFileProviderItemCapabilities {
        if item.isDirectory {
            return [.allowsReading]
        }
        return [.allowsReading]
    }

    var documentSize: NSNumber? {
        guard !item.isDirectory, let size = item.size else {
            return nil
        }
        return NSNumber(value: size)
    }

    var childItemCount: NSNumber? {
        item.isDirectory ? nil : 0
    }

    var creationDate: Date? {
        parseDate(item.createdAt)
    }

    var contentModificationDate: Date? {
        parseDate(item.modifiedAt)
    }

    var itemVersion: NSFileProviderItemVersion {
        let data = Data(item.version.utf8)
        return NSFileProviderItemVersion(contentVersion: data, metadataVersion: data)
    }

    var isUploaded: Bool {
        true
    }

    var isUploading: Bool {
        false
    }

    var isMostRecentVersionDownloaded: Bool {
        true
    }

    private func parseDate(_ value: String?) -> Date? {
        guard let value, !value.isEmpty else {
            return nil
        }
        return ISO8601DateFormatter().date(from: value)
    }
}
