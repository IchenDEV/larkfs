import SwiftUI

struct OverviewView: View {
    let snapshot: DashboardSnapshot
    let isLoading: Bool
    let notice: String?
    let lastUpdatedAt: Date?
    let openConfigDirectory: () -> Void
    let openNativeMountPlan: () -> Void

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 24) {
                heroPanel
                if let notice, !notice.isEmpty {
                    noticePanel
                }
                metricsSection
                healthSection
            }
            .frame(maxWidth: 1040, alignment: .leading)
            .padding(.horizontal, 28)
            .padding(.vertical, 24)
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .navigationTitle("Overview")
    }

    private var heroPanel: some View {
        DashboardPanel {
            VStack(alignment: .leading, spacing: 18) {
                HStack(alignment: .top, spacing: 18) {
                    VStack(alignment: .leading, spacing: 10) {
                        HStack(spacing: 10) {
                            Text("LarkFS Desktop")
                                .font(.system(size: 30, weight: .semibold))
                            StatusPill(
                                title: snapshot.doctor.readinessTitle,
                                tone: snapshot.doctor.ok ? .good : .warning
                            )
                        }

                        Text("A native macOS control plane for LarkFS. Use it to see health, account status, active mounts, and the File Provider rollout path without dropping into the terminal first.")
                            .font(.callout)
                            .foregroundStyle(.secondary)
                            .fixedSize(horizontal: false, vertical: true)
                    }

                    Spacer(minLength: 0)

                    if isLoading {
                        HStack(spacing: 8) {
                            ProgressView()
                                .controlSize(.small)
                            Text("Refreshing snapshot")
                                .font(.callout.weight(.medium))
                        }
                        .padding(.horizontal, 12)
                        .padding(.vertical, 8)
                        .background(.quinary, in: Capsule())
                    }
                }

                HStack(spacing: 10) {
                    ValueBadge(label: "Version", value: snapshot.version.version)
                    ValueBadge(label: "Checks", value: "\(snapshot.doctor.checks.count)")
                    ValueBadge(label: "Healthy Mounts", value: "\(snapshot.healthyMountCount)")
                    if let lastUpdatedText {
                        ValueBadge(label: "Updated", value: lastUpdatedText)
                    }
                }

                HStack(spacing: 12) {
                    Button("Open Config Folder", action: openConfigDirectory)
                        .buttonStyle(DashboardActionButtonStyle())
                    Button("Open Native Mount Plan", action: openNativeMountPlan)
                        .buttonStyle(DashboardActionButtonStyle())
                }
            }
        }
    }

    private var noticePanel: some View {
        DashboardPanel {
            HStack(alignment: .top, spacing: 12) {
                Image(systemName: "info.circle")
                    .foregroundStyle(.secondary)
                VStack(alignment: .leading, spacing: 4) {
                    Text("Snapshot Notice")
                        .font(.headline)
                    Text(notice ?? "")
                        .font(.callout)
                        .foregroundStyle(.secondary)
                        .fixedSize(horizontal: false, vertical: true)
                }
            }
        }
    }

    private var metricsSection: some View {
        VStack(alignment: .leading, spacing: 14) {
            DashboardSectionHeader(
                title: "Core Status",
                subtitle: "The main things you need to know before trying a mount or native integration flow.",
                systemImage: "rectangle.3.group"
            )

            LazyVGrid(
                columns: [
                    GridItem(.flexible(minimum: 260), spacing: 18),
                    GridItem(.flexible(minimum: 260), spacing: 18),
                ],
                alignment: .leading,
                spacing: 18
            ) {
                MetricCard(
                    title: "LarkFS CLI",
                    headline: snapshot.version.version,
                    detail: snapshot.doctor.larkCLI.path ?? "Bundled or external binary unavailable",
                    icon: "terminal"
                )
                MetricCard(
                    title: "Authentication",
                    headline: snapshot.doctor.auth.authenticated ? "Ready" : "Missing",
                    detail: authDetail,
                    icon: "person.crop.circle.badge.checkmark"
                )
                MetricCard(
                    title: "Active Mounts",
                    headline: "\(snapshot.mounts.count)",
                    detail: "\(snapshot.healthyMountCount) healthy",
                    icon: "externaldrive"
                )
                MetricCard(
                    title: "FUSE",
                    headline: snapshot.doctor.fuseCheck?.ok == true ? "Available" : "Unavailable",
                    detail: snapshot.doctor.fuseCheck?.message ?? "Not checked yet",
                    icon: "shippingbox"
                )
            }
        }
    }

    private var healthSection: some View {
        DashboardPanel {
            VStack(alignment: .leading, spacing: 16) {
                HStack(alignment: .center, spacing: 12) {
                    DashboardSectionHeader(
                        title: "Health Checks",
                        subtitle: "Raw checks from the CLI doctor flow, shown inline so failures stay actionable instead of breaking the page.",
                        systemImage: "stethoscope"
                    )
                    Spacer()
                    StatusPill(
                        title: snapshot.doctor.ok ? "Passing" : "Needs attention",
                        tone: snapshot.doctor.ok ? .good : .warning
                    )
                }

                if snapshot.doctor.checks.isEmpty {
                    ContentUnavailableView(
                        "No Health Data Yet",
                        systemImage: "stethoscope",
                        description: Text("Refresh after the bundled CLI is available to populate system checks.")
                    )
                    .frame(maxWidth: .infinity, minHeight: 180)
                } else {
                    VStack(spacing: 12) {
                        ForEach(snapshot.doctor.checks) { check in
                            HealthCheckRow(check: check)
                        }
                    }
                }
            }
        }
    }

    private var authDetail: String {
        if let userName = snapshot.doctor.auth.userName {
            if let identity = snapshot.doctor.auth.identity, !identity.isEmpty {
                return "\(userName) (\(identity))"
            }
            return userName
        }
        return snapshot.doctor.auth.error ?? "Run `larkfs init` to connect your account."
    }

    private var lastUpdatedText: String? {
        guard let lastUpdatedAt else { return nil }
        return lastUpdatedAt.formatted(date: .omitted, time: .shortened)
    }
}

private struct HealthCheckRow: View {
    let check: DoctorCheck

    var body: some View {
        HStack(alignment: .top, spacing: 12) {
            Image(systemName: check.ok ? "checkmark.circle.fill" : "exclamationmark.triangle.fill")
                .foregroundStyle(check.ok ? .green : .orange)
                .font(.title3)

            VStack(alignment: .leading, spacing: 4) {
                Text(check.message)
                    .font(.body.weight(.medium))
                if let hint = check.hint, !hint.isEmpty {
                    Text(hint)
                        .font(.callout)
                        .foregroundStyle(.secondary)
                }
            }

            Spacer(minLength: 0)

            StatusPill(
                title: check.ok ? "OK" : "Fix",
                tone: check.ok ? .good : .warning
            )
        }
        .padding(14)
        .background(.quinary, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
    }
}
