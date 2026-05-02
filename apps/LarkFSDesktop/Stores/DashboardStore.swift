import AppKit
import Foundation

@MainActor
final class DashboardStore: ObservableObject {
    @Published private(set) var snapshot = DashboardSnapshot.empty
    @Published private(set) var isLoading = false
    @Published private(set) var hasLoaded = false
    @Published private(set) var lastUpdatedAt: Date?
    @Published private(set) var nativeDomainStatus = NativeDomainStatus.unknown
    @Published private(set) var isNativeDomainActionRunning = false
    @Published var lastNotice: String?

    private let service: LarkFSCLIService
    private let nativeDomainService = NativeFileProviderDomainService()

    init(service: LarkFSCLIService) {
        self.service = service
    }

    func refresh() async {
        guard !isLoading else { return }

        isLoading = true
        defer {
            isLoading = false
            hasLoaded = true
        }

        async let snapshotResult = service.loadSnapshot()
        async let domainStatus = nativeDomainService.status()

        let result = await snapshotResult
        snapshot = result.snapshot
        nativeDomainStatus = await domainStatus
        lastNotice = result.notice
        lastUpdatedAt = Date()
    }

    func registerNativeDomain() async {
        await runNativeDomainAction {
            try await nativeDomainService.registerDomain()
        }
    }

    func removeNativeDomain() async {
        await runNativeDomainAction {
            try await nativeDomainService.removeDomain()
        }
    }

    func refreshNativeDomainStatus() async {
        nativeDomainStatus = await nativeDomainService.status()
    }

    func openConfigDirectory() {
        NSWorkspace.shared.open(BundlePaths.configDirectory)
    }

    func openNativeMountPlan() {
        guard let url = BundlePaths.nativeMountPlanURL else { return }
        NSWorkspace.shared.open(url)
    }

    private func runNativeDomainAction(_ action: () async throws -> Void) async {
        guard !isNativeDomainActionRunning else { return }
        isNativeDomainActionRunning = true
        defer { isNativeDomainActionRunning = false }

        do {
            try await action()
            nativeDomainStatus = await nativeDomainService.status()
            await refresh()
        } catch {
            lastNotice = "File Provider: \(error.localizedDescription)"
            nativeDomainStatus = await nativeDomainService.status()
        }
    }
}
