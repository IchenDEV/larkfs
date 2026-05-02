import Foundation

struct DashboardLoadResult {
    let snapshot: DashboardSnapshot
    let notice: String?
}

enum SidebarSection: String, CaseIterable, Identifiable {
    case overview
    case mounts
    case nativeMount

    var id: String { rawValue }

    var title: String {
        switch self {
        case .overview:
            return "Overview"
        case .mounts:
            return "Mounts"
        case .nativeMount:
            return "Native API"
        }
    }

    var systemImage: String {
        switch self {
        case .overview:
            return "rectangle.3.group"
        case .mounts:
            return "externaldrive"
        case .nativeMount:
            return "macwindow.badge.plus"
        }
    }

    var detail: String {
        switch self {
        case .overview:
            return "CLI, auth, health"
        case .mounts:
            return "FUSE and WebDAV state"
        case .nativeMount:
            return "File Provider roadmap"
        }
    }
}

struct DashboardSnapshot {
    var version: VersionResponse
    var doctor: DoctorResponse
    var mounts: [MountInfo]
    var nativeCapability: NativeMountCapability

    var healthyMountCount: Int {
        mounts.filter { $0.status == "healthy" }.count
    }

    var failedCheckCount: Int {
        doctor.checks.filter { !$0.ok }.count
    }

    static let empty = DashboardSnapshot(
        version: VersionResponse(version: "unknown", commit: "unknown", date: "unknown"),
        doctor: DoctorResponse(
            ok: false,
            larkCLI: CLIStatus(found: false, path: nil, error: "larkfs binary not found"),
            auth: AuthStatus(authenticated: false, userName: nil, identity: nil, error: nil),
            checks: []
        ),
        mounts: [],
        nativeCapability: NativeMountCapability.current()
    )
}

struct VersionResponse: Decodable {
    let version: String
    let commit: String
    let date: String
}

struct DoctorResponse: Decodable {
    let ok: Bool
    let larkCLI: CLIStatus
    let auth: AuthStatus
    let checks: [DoctorCheck]

    enum CodingKeys: String, CodingKey {
        case ok
        case larkCLI = "lark_cli"
        case auth
        case checks
    }

    var fuseCheck: DoctorCheck? {
        checks.first(where: { $0.name == "fuse" })
    }

    var readinessTitle: String {
        if checks.isEmpty {
            return "Health data unavailable"
        }
        if ok {
            return "Ready for use"
        }
        let failures = checks.filter { !$0.ok }.count
        return failures == 1 ? "1 issue needs attention" : "\(failures) issues need attention"
    }
}

struct CLIStatus: Decodable {
    let found: Bool
    let path: String?
    let error: String?
}

struct AuthStatus: Decodable {
    let authenticated: Bool
    let userName: String?
    let identity: String?
    let error: String?
}

struct DoctorCheck: Identifiable, Decodable {
    let name: String
    let status: String
    let ok: Bool
    let message: String
    let hint: String?

    var id: String { name }
}

struct MountInfo: Identifiable, Decodable {
    let pid: Int
    let mountpoint: String
    let backend: String
    let startedAt: String
    let domains: [String]
    let logFile: String?
    let uptime: String
    let status: String

    var id: String { mountpoint }
}
