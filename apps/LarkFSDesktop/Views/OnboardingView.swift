import SwiftUI

struct OnboardingView: View {
    let snapshot: DashboardSnapshot
    let domainStatus: NativeDomainStatus
    let isLoading: Bool
    let notice: String?
    let lastUpdatedAt: Date?
    let startGuidedSetup: () -> Void
    let refresh: () -> Void
    let openConfigDirectory: () -> Void
    let showMounts: () -> Void
    let showNativeMount: () -> Void

    private var setupSteps: [OnboardingStep] {
        [
            OnboardingStep(
                id: "bridge",
                title: "LarkFS Bridge",
                detail: bridgeDetail,
                icon: "app.connected.to.app.below.fill",
                isComplete: snapshot.nativeCapability.bridgeBinaryFound,
                actionTitle: "Refresh",
                actionIcon: "arrow.clockwise",
                action: .refresh
            ),
            OnboardingStep(
                id: "lark-cli",
                title: "Lark CLI",
                detail: larkCLIDetail,
                icon: "terminal",
                isComplete: snapshot.doctor.larkCLI.found,
                actionTitle: "Run Guided Setup",
                actionIcon: "play.fill",
                action: .guidedSetup
            ),
            OnboardingStep(
                id: "auth",
                title: "Account Login",
                detail: authDetail,
                icon: "person.crop.circle.badge.checkmark",
                isComplete: snapshot.doctor.auth.authenticated,
                actionTitle: "Run Guided Setup",
                actionIcon: "play.fill",
                action: .guidedSetup
            ),
            OnboardingStep(
                id: "access-path",
                title: "Access Path",
                detail: accessPathDetail,
                icon: "externaldrive.connected.to.line.below",
                isComplete: snapshot.mounts.isEmpty == false || domainStatus.registered,
                actionTitle: snapshot.mounts.isEmpty ? "View Mounts" : "View Native API",
                actionIcon: snapshot.mounts.isEmpty ? "externaldrive" : "macwindow.badge.plus",
                action: snapshot.mounts.isEmpty ? .mounts : .nativeMount
            ),
        ]
    }

    private var completedCount: Int { setupSteps.filter(\.isComplete).count }
    private var nextStep: OnboardingStep? { setupSteps.first(where: { !$0.isComplete }) }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 20) {
                heroPanel
                if let notice, !notice.isEmpty {
                    noticePanel
                }
                nextActionPanel
                checklistSection
                accessOptionsSection
            }
            .frame(maxWidth: 1080, alignment: .leading)
            .padding(.horizontal, 24)
            .padding(.vertical, 18)
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .navigationTitle("Get Started")
    }

    private var heroPanel: some View {
        DashboardPanel {
            VStack(alignment: .leading, spacing: 18) {
                HStack(alignment: .top, spacing: 16) {
                    VStack(alignment: .leading, spacing: 10) {
                        HStack(alignment: .center, spacing: 10) {
                            Text("Set Up LarkFS")
                                .font(.system(size: 30, weight: .semibold))
                            StatusPill(
                                title: completedCount == setupSteps.count ? "Ready" : "\(completedCount)/\(setupSteps.count) ready",
                                tone: completedCount == setupSteps.count ? .good : .warning
                            )
                        }

                        Text("Finish the few checks needed before using LarkFS from Finder, WebDAV, or the command line.")
                            .font(.callout)
                            .foregroundStyle(.secondary)
                            .fixedSize(horizontal: false, vertical: true)
                    }

                    Spacer(minLength: 0)

                    if isLoading {
                        HStack(spacing: 8) {
                            ProgressView()
                                .controlSize(.small)
                            Text("Refreshing")
                                .font(.callout.weight(.medium))
                        }
                        .padding(.horizontal, 12)
                        .padding(.vertical, 8)
                        .background(.quinary, in: Capsule())
                    }
                }

                ProgressView(value: Double(completedCount), total: Double(setupSteps.count))
                    .tint(.accentColor)

                HStack(alignment: .center, spacing: 10) {
                    Button(action: startGuidedSetup) {
                        Label("Run Guided Setup", systemImage: "play.fill")
                    }
                    .buttonStyle(DashboardActionButtonStyle(prominent: true))

                    Button(action: refresh) {
                        Label("Refresh", systemImage: "arrow.clockwise")
                    }
                    .buttonStyle(DashboardActionButtonStyle())

                    if let lastUpdatedText {
                        ValueBadge(label: "Updated", value: lastUpdatedText)
                    }
                }
            }
        }
    }

    private var noticePanel: some View {
        DashboardPanel {
            HStack(alignment: .top, spacing: 12) {
                Image(systemName: "info.circle")
                    .foregroundStyle(.secondary)
                    .frame(width: 24)
                VStack(alignment: .leading, spacing: 4) {
                    Text("Setup Notice")
                        .font(.headline)
                    Text(notice ?? "")
                        .font(.callout)
                        .foregroundStyle(.secondary)
                        .fixedSize(horizontal: false, vertical: true)
                }
            }
        }
    }

    private var nextActionPanel: some View {
        DashboardPanel {
            HStack(alignment: .center, spacing: 14) {
                Image(systemName: nextStep?.icon ?? "checkmark.seal.fill")
                    .font(.title3)
                    .foregroundStyle(nextStep == nil ? .green : .orange)
                    .frame(width: 34, height: 34)
                    .background(.quinary, in: RoundedRectangle(cornerRadius: 8, style: .continuous))

                VStack(alignment: .leading, spacing: 8) {
                    Text(nextStep == nil ? "Setup Complete" : "Next: \(nextStep?.title ?? "")")
                        .font(.headline)
                    Text(nextStep?.detail ?? "Core desktop checks are ready. Choose a mount method when you need one.")
                        .font(.callout)
                        .foregroundStyle(.secondary)
                        .fixedSize(horizontal: false, vertical: true)
                }

                Spacer(minLength: 16)

                if let nextStep {
                    Button {
                        run(nextStep.action)
                    } label: {
                        Label(nextStep.actionTitle, systemImage: nextStep.actionIcon)
                    }
                    .buttonStyle(DashboardActionButtonStyle(prominent: true))
                }
            }
        }
    }

    private var checklistSection: some View {
        VStack(alignment: .leading, spacing: 14) {
            DashboardSectionHeader(
                title: "Setup Checklist",
                subtitle: "The app checks the local bridge, upstream Lark CLI, account login, and the first access path.",
                systemImage: "checklist"
            )

            VStack(spacing: 10) {
                ForEach(Array(setupSteps.enumerated()), id: \.element.id) { index, step in
                    OnboardingStepRow(index: index + 1, step: step) {
                        run(step.action)
                    }
                }
            }
        }
    }

    private var accessOptionsSection: some View {
        VStack(alignment: .leading, spacing: 14) {
            DashboardSectionHeader(
                title: "Choose An Access Path",
                subtitle: "WebDAV is the fastest first run. FUSE is for local mount semantics. File Provider is the native Finder path.",
                systemImage: "point.3.connected.trianglepath.dotted"
            )

            LazyVGrid(
                columns: [
                    GridItem(.flexible(minimum: 260), spacing: 14),
                    GridItem(.flexible(minimum: 260), spacing: 14),
                ],
                alignment: .leading,
                spacing: 14
            ) {
                AccessOptionCard(
                    title: "WebDAV",
                    detail: "Start a local WebDAV server and connect from Finder without kernel extensions.",
                    command: "larkfs serve --port 8080",
                    icon: "network",
                    actionTitle: "View Mounts",
                    action: showMounts
                )
                AccessOptionCard(
                    title: "FUSE",
                    detail: "Use a local directory mount when macFUSE or Fuse-T is available.",
                    command: "larkfs mount ~/lark",
                    icon: "externaldrive",
                    actionTitle: "View Mounts",
                    action: showMounts
                )
                AccessOptionCard(
                    title: "File Provider",
                    detail: "Register the Finder domain when the packaged extension is available.",
                    command: "Native API",
                    icon: "macwindow.badge.plus",
                    actionTitle: "Open Native API",
                    action: showNativeMount
                )
                AccessOptionCard(
                    title: "Config",
                    detail: "Open the local LarkFS config directory for logs, mappings, and generated setup helpers.",
                    command: "~/.larkfs",
                    icon: "folder",
                    actionTitle: "Open Folder",
                    action: openConfigDirectory
                )
            }
        }
    }

    private func run(_ action: OnboardingAction) {
        switch action {
        case .guidedSetup:
            startGuidedSetup()
        case .refresh:
            refresh()
        case .mounts:
            showMounts()
        case .nativeMount:
            showNativeMount()
        }
    }

    private var bridgeDetail: String {
        snapshot.nativeCapability.binaryURL?.path ?? "No bundled or external larkfs bridge was found."
    }

    private var larkCLIDetail: String {
        snapshot.doctor.larkCLI.path ?? "Install the upstream Lark CLI with npm, then run guided setup."
    }

    private var authDetail: String {
        if let userName = snapshot.doctor.auth.userName {
            return userName
        }
        return snapshot.doctor.auth.error ?? "Run guided setup to connect your Lark account."
    }

    private var accessPathDetail: String {
        if domainStatus.registered {
            return "Finder domain registered."
        }
        if snapshot.mounts.isEmpty == false {
            return "\(snapshot.mounts.count) active mount session."
        }
        return "Choose WebDAV, FUSE, or File Provider after setup."
    }

    private var lastUpdatedText: String? { lastUpdatedAt?.formatted(date: .omitted, time: .shortened) }
}
