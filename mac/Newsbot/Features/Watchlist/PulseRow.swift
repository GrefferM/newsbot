import NewsbotCore
import SwiftUI

struct PulseRow: View {
    let coin: PulseCoin
    let keeping: Keeping

    @Environment(AppStore.self) private var store
    @State private var hovering = false

    var body: some View {
        VStack(alignment: .leading, spacing: 3) {
            LazyVGrid(columns: tableColumns, spacing: 0) {
                HStack(spacing: 5) {
                    Circle()
                        .fill(mark)
                        .frame(width: 6, height: 6)
                        .help(state)

                    Text(coin.asset)
                        .font(.system(size: 13, weight: .semibold))
                        .lineLimit(1)
                        .truncationMode(.tail)

                    if let trend = coin.trend {
                        Image(systemName: (coin.trendMove ?? 0) < 0 ? "arrow.down.right" : "arrow.up.right")
                            .font(.system(size: 9, weight: .bold))
                            .foregroundStyle((coin.trendMove ?? 0) < 0 ? Theme.redText : Theme.greenText)
                            .help(trend)
                    }
                }

                Text(coin.price > 0 ? Format.price(coin.price) : "—")
                    .lineLimit(1)

                Change(value: known ? coin.change15m : nil)
                Change(value: known ? coin.change1h : nil)
                Change(value: known ? coin.change4h : nil)

                Text(known ? String(format: "×%.1f", coin.volumeSurge) : "—")
                    .foregroundStyle(coin.volumeSurge >= 2 ? Theme.orangeText : Theme.text)
                    .help("Обіг останньої години проти звичайної: \(Format.usd(coin.volumeHour))")

                Text(known ? String(Format.percent(coin.buyShare).dropFirst()) : "—")
                    .foregroundStyle(!known ? Theme.text : coin.buyShare >= 0.5 ? Theme.greenText : Theme.redText)
                    .help("Частка ринкових покупок в обігу останньої години")
            }
            .font(Theme.mono)

            if let caption = coin.caption {
                Text(caption)
                    .font(.system(size: 11))
                    .foregroundStyle(coin.headlines.isEmpty ? Theme.caption : Theme.secondary)
                    .lineLimit(1)
                    .padding(.leading, 11)
                    .help(coin.headlines.map(\.title).joined(separator: "\n"))
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background(coin.active ? Theme.orange.opacity(0.07) : .clear)
        .overlay(alignment: .trailing) {
            if keeping != .none {
                RowAction(asset: coin.asset, browsing: keeping == .browsing, visible: hovering || keeping == .browsing)
                    .padding(4)
                    .background(Theme.window.opacity(hovering || keeping == .browsing ? 1 : 0), in: Circle())
                    .padding(.trailing, 8)
            }
        }
        .onHover { hovering = $0 }
        .overlay(alignment: .bottom) {
            Theme.faintRule
                .frame(height: 1)
        }
    }

    private var known: Bool {
        !coin.contract.isEmpty
    }

    private var listed: WatchCoin? {
        store.coin(coin.asset)
    }

    private var mark: Color {
        guard let listed else {
            return .clear
        }

        return listed.tagged ? Theme.orange : Theme.raised
    }

    private var state: String {
        guard let listed else {
            return ""
        }

        return listed.tagName
    }
}
