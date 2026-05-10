import SwiftUI

enum OnboardingAction {
    case guidedSetup
    case refresh
    case mounts
    case nativeMount
}

struct OnboardingStep: Identifiable {
    let id: String
    let title: String
    let detail: String
    let icon: String
    let isComplete: Bool
    let actionTitle: String
    let actionIcon: String
    let action: OnboardingAction
}

struct OnboardingStepRow: View {
    let index: Int
    let step: OnboardingStep
    let runAction: () -> Void

    var body: some View {
        DashboardPanel {
            HStack(alignment: .center, spacing: 14) {
                statusIcon

                VStack(alignment: .leading, spacing: 8) {
                    HStack(spacing: 8) {
                        Label(step.title, systemImage: step.icon)
                            .font(.headline)
                        StatusPill(
                            title: step.isComplete ? "Done" : "Next",
                            tone: step.isComplete ? .good : .warning
                        )
                    }

                    Text(step.detail)
                        .font(.callout)
                        .foregroundStyle(.secondary)
                        .textSelection(.enabled)
                        .fixedSize(horizontal: false, vertical: true)
                }

                Spacer(minLength: 0)

                if !step.isComplete {
                    Button(action: runAction) {
                        Label(step.actionTitle, systemImage: step.actionIcon)
                    }
                    .buttonStyle(DashboardActionButtonStyle(prominent: step.action == .guidedSetup))
                }
            }
            .frame(minHeight: 58)
        }
    }

    private var statusIcon: some View {
        ZStack {
            Circle()
                .fill(step.isComplete ? Color.green.opacity(0.16) : Color.orange.opacity(0.16))
            Image(systemName: step.isComplete ? "checkmark" : "\(index).circle")
                .font(.callout.weight(.semibold))
                .foregroundStyle(step.isComplete ? .green : .orange)
        }
        .frame(width: 30, height: 30)
    }
}

struct AccessOptionCard: View {
    let title: String
    let detail: String
    let command: String
    let icon: String
    let actionTitle: String
    let action: () -> Void

    var body: some View {
        DashboardPanel {
            VStack(alignment: .leading, spacing: 14) {
                Label(title, systemImage: icon)
                    .font(.headline)
                Text(detail)
                    .font(.callout)
                    .foregroundStyle(.secondary)
                    .fixedSize(horizontal: false, vertical: true)
                CommandSnippet(command: command)
                Spacer(minLength: 0)
                Button(action: action) {
                    Label(actionTitle, systemImage: "arrow.right")
                }
                .buttonStyle(DashboardActionButtonStyle())
            }
            .frame(maxWidth: .infinity, minHeight: 196, alignment: .leading)
        }
    }
}
