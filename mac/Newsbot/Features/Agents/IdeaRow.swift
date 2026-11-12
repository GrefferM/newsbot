import NewsbotCore
import SwiftUI

struct IdeaRow: View {
    let idea: Idea

    @Environment(\.openWindow) private var openWindow
    @Environment(Navigator.self) private var navigator

    var body: some View {
        Button {
            navigator.show(coin: idea.asset)
            openWindow(id: "coin")
        } label: {
            VStack(alignment: .leading, spacing: 3) {
                HStack(spacing: 8) {
                    Text(idea.asset)
                        .font(.system(size: 13, weight: .semibold))

                    Pill(text: idea.direction, foreground: tint, background: tint.opacity(0.14))

                    Spacer()

                    Image(systemName: "chevron.right")
                        .font(.system(size: 10, weight: .semibold))
                        .foregroundStyle(Theme.caption)
                }

                if let facts = idea.facts {
                    Text(facts)
                        .font(.system(size: 11, design: .monospaced))
                        .foregroundStyle(Theme.caption)
                }

                Text(idea.why)
                    .font(.system(size: 12))
                    .fixedSize(horizontal: false, vertical: true)
                    .multilineTextAlignment(.leading)

                if let event = idea.event, !event.isEmpty {
                    Text("Подія: \(event)")
                        .font(.system(size: 11))
                        .foregroundStyle(Theme.secondary)
                        .fixedSize(horizontal: false, vertical: true)
                        .multilineTextAlignment(.leading)
                }

                Text("Скасовує: \(idea.cancel)")
                    .font(.system(size: 11))
                    .foregroundStyle(Theme.secondary)
                    .fixedSize(horizontal: false, vertical: true)
                    .multilineTextAlignment(.leading)
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 8)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(Theme.card, in: RoundedRectangle(cornerRadius: 10))
            .overlay(RoundedRectangle(cornerRadius: 10).stroke(tint.opacity(0.35)))
            .contentShape(RoundedRectangle(cornerRadius: 10))
        }
        .buttonStyle(.plain)
        .help("Відкрити сторінку \(idea.asset): там Ризик-менеджер складе план із числами")
    }

    private var tint: Color {
        idea.long ? Theme.greenText : Theme.redText
    }
}
