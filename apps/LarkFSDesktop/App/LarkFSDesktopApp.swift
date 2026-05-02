import SwiftUI

@main
struct LarkFSDesktopApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate
    @StateObject private var store = DashboardStore(service: LarkFSCLIService())

    var body: some Scene {
        WindowGroup("LarkFS Desktop") {
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

                Button("Open Native Mount Plan") {
                    store.openNativeMountPlan()
                }
            }
        }
    }
}
