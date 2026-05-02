import SwiftUI

struct DashboardPanel<Content: View>: View {
    @ViewBuilder let content: Content

    var body: some View {
        content
            .padding(18)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background {
                RoundedRectangle(cornerRadius: 18, style: .continuous)
                    .fill(.regularMaterial)
            }
            .overlay {
                RoundedRectangle(cornerRadius: 18, style: .continuous)
                    .strokeBorder(Color.white.opacity(0.06))
            }
    }
}

struct DashboardSectionHeader: View {
    let title: String
    let subtitle: String?
    let systemImage: String

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Label(title, systemImage: systemImage)
                .font(.headline)

            if let subtitle, !subtitle.isEmpty {
                Text(subtitle)
                    .font(.callout)
                    .foregroundStyle(.secondary)
            }
        }
    }
}

struct MetricCard: View {
    let title: String
    let headline: String
    let detail: String
    let icon: String

    var body: some View {
        DashboardPanel {
            VStack(alignment: .leading, spacing: 14) {
                Label(title, systemImage: icon)
                    .font(.headline)

                Text(headline)
                    .font(.system(size: 30, weight: .semibold))
                    .lineLimit(1)

                Text(detail)
                    .foregroundStyle(.secondary)
                    .fixedSize(horizontal: false, vertical: true)
                    .textSelection(.enabled)
            }
            .frame(maxWidth: .infinity, minHeight: 154, alignment: .leading)
        }
    }
}

struct PropertyRow: View {
    let title: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(title)
                .font(.caption)
                .foregroundStyle(.secondary)
            Text(value)
                .textSelection(.enabled)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

struct CommandSnippet: View {
    let command: String

    var body: some View {
        Text(command)
            .font(.system(.callout, design: .monospaced))
            .textSelection(.enabled)
            .padding(.horizontal, 12)
            .padding(.vertical, 10)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(.quinary, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
    }
}

struct StatusPill: View {
    enum Tone {
        case good
        case warning
        case neutral
    }

    let title: String
    let tone: Tone

    var body: some View {
        Text(title)
            .font(.callout.weight(.medium))
            .padding(.horizontal, 10)
            .padding(.vertical, 6)
            .background(backgroundColor, in: Capsule())
            .foregroundStyle(foregroundColor)
    }

    private var backgroundColor: Color {
        switch tone {
        case .good:
            return Color.green.opacity(0.16)
        case .warning:
            return Color.orange.opacity(0.18)
        case .neutral:
            return Color.secondary.opacity(0.12)
        }
    }

    private var foregroundColor: Color {
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

struct ValueBadge: View {
    let label: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(label)
                .font(.caption)
                .foregroundStyle(.secondary)
            Text(value)
                .font(.callout.weight(.semibold))
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 8)
        .background(.quinary, in: RoundedRectangle(cornerRadius: 10, style: .continuous))
    }
}

struct DashboardActionButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
            .background(.quinary.opacity(configuration.isPressed ? 0.7 : 1), in: RoundedRectangle(cornerRadius: 10, style: .continuous))
    }
}
