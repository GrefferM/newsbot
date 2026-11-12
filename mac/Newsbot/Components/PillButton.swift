import SwiftUI

struct PillButton: View {
    let icon: String
    let title: String
    var active = false
    let action: () async -> Void

    var body: some View {
        Button {
            Task {
                await action()
            }
        } label: {
            HStack(spacing: 4) {
                Image(systemName: icon)
                    .font(.system(size: 9, weight: .semibold))

                Text(title)
                    .font(.system(size: 11))
            }
            .padding(.horizontal, 7)
            .padding(.vertical, 2)
            .foregroundStyle(active ? Theme.blueText : Theme.secondary)
            .background(active ? Theme.accent.opacity(0.18) : .clear, in: Capsule())
            .overlay(Capsule().stroke(active ? .clear : Color.white.opacity(0.14)))
            .contentShape(Capsule())
        }
        .buttonStyle(.plain)
        .accessibilityLabel(title)
    }
}
