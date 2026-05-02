import Foundation
import FileProvider

enum LarkFSFileProviderError {
    static func noSuchItem(_ identifier: NSFileProviderItemIdentifier) -> NSError {
        NSError(
            domain: NSFileProviderErrorDomain,
            code: NSFileProviderError.noSuchItem.rawValue,
            userInfo: [NSLocalizedDescriptionKey: "LarkFS item not found: \(identifier.rawValue)"]
        )
    }

    static func readOnly(_ message: String) -> NSError {
        NSError(
            domain: NSFileProviderErrorDomain,
            code: NSFileProviderError.cannotSynchronize.rawValue,
            userInfo: [NSLocalizedDescriptionKey: message]
        )
    }

    static func bridge(_ error: Error) -> NSError {
        NSError(
            domain: NSCocoaErrorDomain,
            code: NSXPCConnectionReplyInvalid,
            userInfo: [
                NSLocalizedDescriptionKey: error.localizedDescription,
                NSUnderlyingErrorKey: error,
            ]
        )
    }
}

func larkFSProgress(completed: Int64 = 0, total: Int64 = 1) -> Progress {
    let progress = Progress(totalUnitCount: total)
    progress.completedUnitCount = completed
    return progress
}

final class SendableBox<Value>: @unchecked Sendable {
    let value: Value

    init(_ value: Value) {
        self.value = value
    }
}
