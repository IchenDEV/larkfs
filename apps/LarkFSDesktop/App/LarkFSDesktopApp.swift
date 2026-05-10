import SwiftUI

@main
struct LarkFSDesktopApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate
    @StateObject private var store = DashboardStore(service: LarkFSCLIService())

    var body: some Scene {
        WindowGroup("LarkFS Desktop", id: "main") {
            ContentView(store: store)
        }
        .defaultSize(width: 1240, height: 860)
        .commands {
            CommandMenu("LarkFS") {
                Button("Refresh") {
                    Task {
                        await store.refresh()
                    }
                }
                .keyboardShortcut("r")

                Button("Open Config Folder") {
                    store.openConfigDirectory()
                }

                Button("Start Guided Setup") {
                    store.startGuidedSetup()
                }
                .keyboardShortcut("i", modifiers: [.command, .shift])

                Button("Open Native Mount Plan") {
                    store.openNativeMountPlan()
                }
            }
        }

        MenuBarExtra {
            MenuBarStatusMenu(store: store)
        } label: {
            Image(systemName: store.syncStatus.systemImage)
                .accessibilityLabel("LarkFS Sync Status")
                .task {
                    store.startPeriodicRefresh()
                    if !store.hasLoaded {
                        await store.refresh()
                    }
                }
        }
        .menuBarExtraStyle(.window)
    }
}
