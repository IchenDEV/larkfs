import Foundation
#if canImport(FileProvider)
import FileProvider
#endif

struct NativeDomainStatus: Sendable {
    let supported: Bool
    let registered: Bool
    let message: String

    static let unavailable = NativeDomainStatus(
        supported: false,
        registered: false,
        message: "File Provider runtime is unavailable on this system."
    )

    static let unknown = NativeDomainStatus(
        supported: false,
        registered: false,
        message: "Domain status has not been checked yet."
    )
}

struct NativeFileProviderDomainService: Sendable {
    func status() async -> NativeDomainStatus {
        #if canImport(FileProvider)
        do {
            let found = try await isDomainRegistered()
            return NativeDomainStatus(
                supported: true,
                registered: found,
                message: found ? "LarkFS is registered with File Provider." : "LarkFS is not registered with File Provider yet."
            )
        } catch {
            return NativeDomainStatus(
                supported: true,
                registered: false,
                message: error.localizedDescription
            )
        }
        #else
        return .unavailable
        #endif
    }

    func registerDomain() async throws {
        #if canImport(FileProvider)
        try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Void, Error>) in
            NSFileProviderManager.add(makeDomain()) { error in
                if let error {
                    continuation.resume(throwing: error)
                } else {
                    continuation.resume()
                }
            }
        }
        #endif
    }

    func removeDomain() async throws {
        #if canImport(FileProvider)
        try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Void, Error>) in
            NSFileProviderManager.remove(makeDomain()) { error in
                if let error {
                    continuation.resume(throwing: error)
                } else {
                    continuation.resume()
                }
            }
        }
        #endif
    }

    #if canImport(FileProvider)
    private func isDomainRegistered() async throws -> Bool {
        try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Bool, Error>) in
            NSFileProviderManager.getDomainsWithCompletionHandler { domains, error in
                if let error {
                    continuation.resume(throwing: error)
                } else {
                    let found = domains.contains { $0.identifier.rawValue == nativeDomainIdentifier.rawValue }
                    continuation.resume(returning: found)
                }
            }
        }
    }

    private func makeDomain() -> NSFileProviderDomain {
        let domain = NSFileProviderDomain(identifier: nativeDomainIdentifier, displayName: "LarkFS")
        domain.testingModes = [.alwaysEnabled]
        return domain
    }
    #endif
}

#if canImport(FileProvider)
private let nativeDomainIdentifier = NSFileProviderDomainIdentifier("dev.ichen.larkfs")
#endif
