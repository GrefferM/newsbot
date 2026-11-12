import NewsbotCore
import SwiftUI

struct StrategyCard: View {
    let strategy: AutoStrategy

    @Environment(AppStore.self) private var store
    @Environment(\.openWindow) private var openWindow

    private let columns = [
        GridItem(.flexible(), spacing: 16, alignment: .topLeading),
        GridItem(.flexible(), spacing: 16, alignment: .topLeading),
    ]

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            HStack(spacing: 10) {
                Text(strategy.name)
                    .font(.system(size: 16, weight: .semibold))

                Pill(
                    text: strategy.on ? "ПРАЦЮЄ" : "ВИМКНЕНО",
                    foreground: strategy.on ? Theme.greenText : Theme.secondary,
                    background: strategy.on ? Theme.green.opacity(0.18) : Theme.control)

                Spacer()

                PillButton(icon: "chart.xyaxis.line", title: "Стенд") {
                    openWindow(id: "stand", value: strategy.id)
                }

                Toggle(strategy.name, isOn: wanted)
                    .toggleStyle(.switch)
                    .controlSize(.small)
                    .labelsHidden()
                    .tint(Theme.green)
            }

            Text(strategy.about)
                .font(.system(size: 13))
                .foregroundStyle(Theme.secondary)
                .fixedSize(horizontal: false, vertical: true)

            Divider()
                .overlay(Theme.rule)

            LazyVGrid(columns: columns, alignment: .leading, spacing: 12) {
                ForEach(strategy.terms) { term in
                    VStack(alignment: .leading, spacing: 3) {
                        Text(term.name.uppercased())
                            .font(.system(size: 10, weight: .semibold))
                            .foregroundStyle(Theme.caption)

                        Text(term.value)
                            .font(Theme.mono)
                            .fixedSize(horizontal: false, vertical: true)
                    }
                }
            }

            Divider()
                .overlay(Theme.rule)

            Text(strategy.record.line)
                .font(.system(size: 12))
                .foregroundStyle(Theme.secondary)
        }
        .padding(18)
        .background(Theme.card, in: RoundedRectangle(cornerRadius: 10))
    }

    private var wanted: Binding<Bool> {
        Binding {
            strategy.on
        } set: { on in
            Task {
                await store.switchStrategy(strategy.id, on: on)
            }
        }
    }
}
