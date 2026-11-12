import NewsbotCore
import SwiftUI

struct OrdersCard: View {
    let asset: String
    let plans: CoinPlans

    @Environment(AppStore.self) private var store
    @Environment(\.openWindow) private var openWindow

    var body: some View {
        CardFrame(title: "Ордери", note: store.book.venueName) {
            if positions.isEmpty && plans.waiting.isEmpty && settled.isEmpty {
                Remark(
                    text:
                        "Тут чекають плани, які ти поставив на стеження. Коли ціна доходить до порога плану, агент перевіряє умову — і план стає ордером або знімається."
                )
            }

            ForEach(positions) { position in
                Button {
                    openWindow(id: "position", value: position.id)
                } label: {
                    Row(
                        name: "\(position.side.name) \(position.contract) · вхід \(Format.price(position.entry))",
                        value: Format.usd(position.stake),
                        tint: Theme.blueText
                    )
                    .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .help("Відкрити позицію")
            }

            ForEach(plans.waiting) { watch in
                WatchRow(watch: watch, plans: plans)
            }

            ForEach(settled) { watch in
                WatchRow(watch: watch, plans: nil)
            }

            if let failure = plans.failure {
                Text(failure)
                    .font(.system(size: 11))
                    .foregroundStyle(Theme.redText)
            }
        }
        .fixedSize(horizontal: false, vertical: true)
    }

    private var positions: [Position] {
        store.book.openPositions.filter { $0.symbol == asset }
    }

    private var settled: [PlanWatch] {
        plans.settled(last: 5)
    }
}
