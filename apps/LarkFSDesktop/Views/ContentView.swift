import SwiftUI

struct ContentView: View {
    @ObservedObject var store: DashboardStore
    @Environment(\.colorScheme) private var colorScheme
    @SceneStorage("selected-section") private var selectedSectionRawValue = SidebarSection.overview.rawValue

    private var selectedSection: SidebarSection {
        get { SidebarSection(rawValue: selectedSectionRawValue) ?? .overview }
        nonmutating set { selectedSectionRawValue = newValue.rawValue }
    }

    var body: some View {
        NavigationSplitView {
            SidebarView(selection: selectionBinding, snapshot: store.snapshot)
                .navigationSplitViewColumnWidth(min: 180, ideal: 210, max: 240)
        } detail: {
            detailView
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        }
        .navigationSplitViewStyle(.balanced)
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button {
                    Task {
                        await store.refresh()
                    }
                } label: {
                    if store.isLoading {
                        HStack(spacing: 8) {
                            ProgressView()
                                .controlSize(.small)
                            Text("Refreshing…")
                        }
                    } else {
                        Label("Refresh", systemImage: "arrow.clockwise")
                    }
                }
                .disabled(store.isLoading)
            }
        }
        .task {
            if !store.hasLoaded {
                await store.refresh()
            }
        }
        .task(id: colorScheme) {
            AppIconAppearanceService.apply(colorScheme: colorScheme)
        }
    }

    private var selectionBinding: Binding<SidebarSection?> {
        Binding {
            selectedSection
        } set: { newValue in
            selectedSection = newValue ?? .overview
        }
    }

    @ViewBuilder
    private var detailView: some View {
        switch selectedSection {
        case .overview:
            OverviewView(
                snapshot: store.snapshot,
                isLoading: store.isLoading,
                notice: store.lastNotice,
                lastUpdatedAt: store.lastUpdatedAt,
                openConfigDirectory: store.openConfigDirectory,
                openNativeMountPlan: store.openNativeMountPlan
            )
        case .mounts:
            MountsView(mounts: store.snapshot.mounts)
        case .nativeMount:
            NativeMountView(
                capability: store.snapshot.nativeCapability,
                domainStatus: store.nativeDomainStatus,
                isDomainActionRunning: store.isNativeDomainActionRunning,
                openPlan: store.openNativeMountPlan,
                openConfigDirectory: store.openConfigDirectory,
                registerDomain: {
                    Task {
                        await store.registerNativeDomain()
                    }
                },
                removeDomain: {
                    Task {
                        await store.removeNativeDomain()
                    }
                },
                refreshDomainStatus: {
                    Task {
                        await store.refreshNativeDomainStatus()
                    }
                }
            )
        }
    }
}
