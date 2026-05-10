import AppKit
import SwiftUI

struct MenuBarStatusMenu: View {
    @ObservedObject var store: DashboardStore
    @Environment(\.openWindow) private var openWindow

    var body: some View {
        let status = store.syncStatus

        VStack(alignment: .leading, spacing: 14) {
            header(status)
            statusRows
            actions
        }
        .padding(14)
        .frame(width: 320, alignment: .leading)
        .task {
            if !store.hasLoaded {
                await store.refresh()
            }
        }
    }

    private func header(_ status: MenuBarSyncStatus) -> some View {
        HStack(alignment: .center, spacing: 12) {
            Image(systemName: status.systemImage)
                .font(.title3.weight(.semibold))
                .foregroundStyle(status.tintColor)
                .frame(width: 38, height: 38)
                .background(status.tintColor.opacity(0.14), in: RoundedRectangle(cornerRadius: 9, style: .continuous))

            VStack(alignment: .leading, spacing: 3) {
                Text(status.title)
                    .font(.headline)
                Text(status.detail)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
            }

            Spacer(minLength: 8)

            if store.isLoading {
                ProgressView()
                    .controlSize(.small)
            } else {
                Text(status.lastUpdatedText)
                    .font(.caption2.weight(.medium))
                    .foregroundStyle(.secondary)
                    .padding(.horizontal, 8)
                    .frame(height: 24)
                    .background(.quinary, in: Capsule())
            }
        }
    }

    private var statusRows: some View {
        VStack(alignment: .leading, spacing: 8) {
            MenuBarStatusRow(
                title: "Account",
                value: accountValue,
                systemImage: store.snapshot.doctor.auth.authenticated ? "person.crop.circle.badge.checkmark" : "person.crop.circle.badge.exclamationmark"
            )
            MenuBarStatusRow(
                title: "Mounts",
                value: "\(store.snapshot.healthyMountCount)/\(store.snapshot.mounts.count) healthy",
                systemImage: "externaldrive"
            )
            MenuBarStatusRow(
                title: "Finder",
                value: store.nativeDomainStatus.registered ? "Registered" : "Not registered",
                systemImage: "folder.badge.gearshape"
            )
        }
    }

    private var actions: some View {
        VStack(spacing: 8) {
            HStack(spacing: 8) {
                Button {
                    Task {
                        await store.refresh()
                    }
                } label: {
                    Label("Refresh", systemImage: "arrow.clockwise")
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(DashboardActionButtonStyle())
                .keyboardShortcut("r")

                Button {
                    openMainWindow()
                } label: {
                    Label("Open App", systemImage: "macwindow")
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(DashboardActionButtonStyle(prominent: true))
            }

            HStack(spacing: 8) {
                Button {
                    store.startGuidedSetup()
                } label: {
                    Label("Setup", systemImage: "play.fill")
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(DashboardActionButtonStyle())

                Button {
                    store.openConfigDirectory()
                } label: {
                    Label("Config", systemImage: "folder")
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(DashboardActionButtonStyle())
            }

            Divider()

            Button("Quit LarkFS") {
                NSApplication.shared.terminate(nil)
            }
            .buttonStyle(.plain)
            .foregroundStyle(.secondary)
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var accountValue: String {
        if let userName = store.snapshot.doctor.auth.userName, !userName.isEmpty {
            return userName
        }
        return store.snapshot.doctor.auth.authenticated ? "Connected" : "Not connected"
    }

    private func openMainWindow() {
        NSApp.setActivationPolicy(.regular)
        NSApp.activate(ignoringOtherApps: true)
        openWindow(id: "main")
    }
}

private extension MenuBarSyncStatus {
    var tintColor: Color {
        switch tone {
        case .good:
            return .green
        case .warning:
            return .orange
        case .neutral:
            return .secondary
        }
    }
}

private struct MenuBarStatusRow: View {
    let title: String
    let value: String
    let systemImage: String

    var body: some View {
        HStack(spacing: 10) {
            Image(systemName: systemImage)
                .foregroundStyle(.secondary)
                .frame(width: 28, height: 28)
                .background(.quinary, in: RoundedRectangle(cornerRadius: 7, style: .continuous))

            VStack(alignment: .leading, spacing: 2) {
                Text(title)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                Text(value)
                    .font(.callout.weight(.medium))
                    .lineLimit(1)
            }

            Spacer()
        }
        .padding(10)
        .background(.quinary.opacity(0.72), in: RoundedRectangle(cornerRadius: 10, style: .continuous))
    }
}
