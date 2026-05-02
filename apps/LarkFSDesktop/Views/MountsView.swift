import SwiftUI

struct MountsView: View {
    let mounts: [MountInfo]

    private var backendsSummary: String {
        let backends = Array(Set(mounts.map(\.backend))).sorted()
        guard !backends.isEmpty else { return "No active backends" }
        return backends.joined(separator: ", ")
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 24) {
                headerPanel

                if mounts.isEmpty {
                    emptyStatePanel
                } else {
                    mountsSection
                }
            }
            .frame(maxWidth: 1040, alignment: .leading)
            .padding(.horizontal, 28)
            .padding(.vertical, 24)
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .navigationTitle("Mounts")
    }

    private var headerPanel: some View {
        DashboardPanel {
            VStack(alignment: .leading, spacing: 16) {
                HStack(alignment: .top, spacing: 18) {
                    VStack(alignment: .leading, spacing: 10) {
                        Text("Active Mount Sessions")
                            .font(.system(size: 28, weight: .semibold))
                        Text("View every live FUSE or WebDAV session that the desktop control plane can currently see.")
                            .font(.callout)
                            .foregroundStyle(.secondary)
                    }

                    Spacer(minLength: 0)

                    StatusPill(
                        title: mounts.isEmpty ? "No active mounts" : "\(mounts.count) active",
                        tone: mounts.isEmpty ? .neutral : .good
                    )
                }

                HStack(spacing: 10) {
                    ValueBadge(label: "Mounts", value: "\(mounts.count)")
                    ValueBadge(label: "Backends", value: backendsSummary)
                }
            }
        }
    }

    private var emptyStatePanel: some View {
        DashboardPanel {
            VStack(alignment: .leading, spacing: 16) {
                DashboardSectionHeader(
                    title: "Nothing Mounted Yet",
                    subtitle: "Once you start a FUSE mount or WebDAV server, live sessions will appear here.",
                    systemImage: "externaldrive.badge.questionmark"
                )

                CommandSnippet(command: "larkfs mount ~/lark")
                CommandSnippet(command: "larkfs serve --port 8080")
            }
            .frame(maxWidth: .infinity, minHeight: 220, alignment: .leading)
        }
    }

    private var mountsSection: some View {
        VStack(alignment: .leading, spacing: 14) {
            DashboardSectionHeader(
                title: "Session Details",
                subtitle: "Each row tracks backend, uptime, scope, and any log path exposed by the CLI state.",
                systemImage: "list.bullet.rectangle"
            )

            VStack(spacing: 16) {
                ForEach(mounts) { mount in
                    DashboardPanel {
                        VStack(alignment: .leading, spacing: 16) {
                            HStack(alignment: .top, spacing: 12) {
                                VStack(alignment: .leading, spacing: 6) {
                                    Text(mount.mountpoint)
                                        .font(.headline)
                                        .textSelection(.enabled)
                                    Text(mount.backend.uppercased())
                                        .font(.caption.weight(.medium))
                                        .foregroundStyle(.secondary)
                                }

                                Spacer(minLength: 0)

                                StatusPill(
                                    title: mount.status.capitalized,
                                    tone: mount.status == "healthy" ? .good : .warning
                                )
                            }

                            LazyVGrid(
                                columns: [
                                    GridItem(.flexible(minimum: 220), spacing: 14),
                                    GridItem(.flexible(minimum: 220), spacing: 14),
                                ],
                                alignment: .leading,
                                spacing: 14
                            ) {
                                PropertyRow(title: "Uptime", value: mount.uptime)
                                PropertyRow(title: "Domains", value: mount.domains.joined(separator: ", "))
                                PropertyRow(title: "Started At", value: mount.startedAt)
                                if let logFile = mount.logFile, !logFile.isEmpty {
                                    PropertyRow(title: "Log File", value: logFile)
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
