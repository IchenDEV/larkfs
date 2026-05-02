import SwiftUI

struct NativeMountView: View {
    let capability: NativeMountCapability
    let domainStatus: NativeDomainStatus
    let isDomainActionRunning: Bool
    let openPlan: () -> Void
    let openConfigDirectory: () -> Void
    let registerDomain: () -> Void
    let removeDomain: () -> Void
    let refreshDomainStatus: () -> Void

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 24) {
                headerPanel
                capabilitySection
                roadmapSection
            }
            .frame(maxWidth: 1040, alignment: .leading)
            .padding(.horizontal, 28)
            .padding(.vertical, 24)
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .navigationTitle("Native API")
    }

    private var headerPanel: some View {
        DashboardPanel {
            VStack(alignment: .leading, spacing: 16) {
                HStack(alignment: .top, spacing: 18) {
                    VStack(alignment: .leading, spacing: 10) {
                        Text("Finder-Native Mount Path")
                            .font(.system(size: 28, weight: .semibold))
                        Text("This page tracks the path from the existing LarkFS CLI bridge to a proper File Provider integration that behaves more like OneDrive inside Finder.")
                            .font(.callout)
                            .foregroundStyle(.secondary)
                            .fixedSize(horizontal: false, vertical: true)
                    }

                    Spacer(minLength: 0)

                    StatusPill(
                        title: capability.extensionPackaged ? "Extension packaged" : "Host app ready",
                        tone: capability.extensionPackaged ? .good : .warning
                    )
                }

                HStack(spacing: 12) {
                    Button("Open Native Mount Plan", action: openPlan)
                        .buttonStyle(DashboardActionButtonStyle())
                    Button("Open Config Folder", action: openConfigDirectory)
                        .buttonStyle(DashboardActionButtonStyle())
                    Button(domainStatus.registered ? "Remove Finder Domain" : "Register Finder Domain") {
                        domainStatus.registered ? removeDomain() : registerDomain()
                    }
                    .buttonStyle(DashboardActionButtonStyle())
                    .disabled(isDomainActionRunning || !capability.extensionPackaged)
                    Button("Refresh Domain", action: refreshDomainStatus)
                        .buttonStyle(DashboardActionButtonStyle())
                        .disabled(isDomainActionRunning)
                }
            }
        }
    }

    private var capabilitySection: some View {
        VStack(alignment: .leading, spacing: 14) {
            DashboardSectionHeader(
                title: "Current Capability Snapshot",
                subtitle: "These flags describe what the macOS host app can already do and what still needs Xcode-only packaging work.",
                systemImage: "macwindow.badge.plus"
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
                    title: "File Provider Runtime",
                    headline: capability.fileProviderSupported ? "Available" : "Unavailable",
                    detail: capability.domainIdentifier,
                    icon: "shippingbox.circle"
                )
                MetricCard(
                    title: "CLI Bridge",
                    headline: capability.bridgeBinaryFound ? "Ready" : "Missing",
                    detail: capability.binaryURL?.path ?? "No bundled or external bridge binary found",
                    icon: "terminal"
                )
                MetricCard(
                    title: "Plan Document",
                    headline: capability.planDocumentFound ? "Present" : "Missing",
                    detail: capability.workspaceRoot?.path ?? "Workspace root unavailable",
                    icon: "doc.text"
                )
                MetricCard(
                    title: "Extension Packaging",
                    headline: capability.extensionPackaged ? "Ready" : "Pending",
                    detail: "Swift module exists; Xcode target, entitlements, signing, and registration are still pending",
                    icon: "hammer"
                )
                MetricCard(
                    title: "Finder Domain",
                    headline: domainStatus.registered ? "Registered" : "Not Registered",
                    detail: domainStatus.message,
                    icon: "folder.badge.gearshape"
                )
            }
        }
    }

    private var roadmapSection: some View {
        VStack(alignment: .leading, spacing: 14) {
            DashboardSectionHeader(
                title: "Rollout Notes",
                subtitle: "What this stage covers today, plus the steps still needed to reach a Finder-grade experience.",
                systemImage: "list.bullet.rectangle.portrait"
            )

            DashboardPanel {
                VStack(alignment: .leading, spacing: 14) {
                    ForEach(Array(capability.notes.enumerated()), id: \.offset) { index, note in
                        HStack(alignment: .top, spacing: 12) {
                            Text("\(index + 1)")
                                .font(.callout.weight(.semibold))
                                .frame(width: 24, height: 24)
                                .background(.quinary, in: Circle())
                            Text(note)
                                .fixedSize(horizontal: false, vertical: true)
                        }
                    }
                }
            }
        }
    }
}
