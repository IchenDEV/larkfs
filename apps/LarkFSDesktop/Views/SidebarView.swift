import SwiftUI

struct SidebarView: View {
    @Binding var selection: SidebarSection?
    let snapshot: DashboardSnapshot

    var body: some View {
        List(selection: $selection) {
            ForEach(SidebarSection.allCases) { section in
                HStack(spacing: 10) {
                    Image(systemName: section.systemImage)
                        .foregroundStyle(.secondary)
                        .frame(width: 16)

                    VStack(alignment: .leading, spacing: 2) {
                        Text(section.title)
                            .lineLimit(1)

                        Text(detail(for: section))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .lineLimit(1)
                    }
                }
                .tag(section)
            }
        }
        .navigationTitle("LarkFS Desktop")
        .listStyle(.sidebar)
    }

    private func detail(for section: SidebarSection) -> String {
        switch section {
        case .overview:
            if snapshot.doctor.checks.isEmpty {
                return "No snapshot yet"
            }
            if snapshot.doctor.ok {
                return "All checks passing"
            }
            return snapshot.failedCheckCount == 1 ? "1 issue to fix" : "\(snapshot.failedCheckCount) issues to fix"
        case .mounts:
            return snapshot.mounts.isEmpty ? "No active sessions" : "\(snapshot.healthyMountCount) healthy"
        case .nativeMount:
            return snapshot.nativeCapability.extensionPackaged ? "Extension packaged" : "Host app ready"
        }
    }
}
