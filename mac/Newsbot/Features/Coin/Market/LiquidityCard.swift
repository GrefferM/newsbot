import Charts
import NewsbotCore
import SwiftUI

struct LiquidityCard: View {
    let contract: String
    let market: CoinMarket

    @Environment(AppStore.self) private var store

    var body: some View {
        CardFrame(title: "Ліквідність", note: market.bookFailed ? "недоступна" : "перп · наживо") {
            if let liquidity = market.liquidity {
                Row(name: "Спред", value: share(liquidity.spread), tint: Theme.text)

                DepthChart(curve: liquidity.curve)

                BalanceBar(label: "±1%", bid: liquidity.bidNear, ask: liquidity.askNear)
                BalanceBar(label: "±2%", bid: liquidity.bidFar, ask: liquidity.askFar)

                Row(name: "Найбільша заявка", value: walls(liquidity), tint: Theme.text)
                Row(
                    name: "Прослизання на \(Format.usd(store.book.probe))", value: slippage(liquidity),
                    tint: Theme.orangeText)
            } else {
                Text("Читаю стакан…")
                    .font(.system(size: 12))
                    .foregroundStyle(Theme.secondary)
            }
        }
        .task(id: contract) {
            await market.followBook(contract: contract) {
                store.book.probe
            }
        }
    }

    private func share(_ fraction: Double) -> String {
        Format.percent(fraction, digits: 2).dropFirst().description
    }

    private func slippage(_ liquidity: Liquidity) -> String {
        let sell = liquidity.slippage.map(share) ?? "не вистачає"
        let buy = liquidity.buySlippage.map(share) ?? "не вистачає"

        return "продаж \(sell) · купівля \(buy)"
    }

    private func walls(_ liquidity: Liquidity) -> String {
        [liquidity.bidWall, liquidity.askWall]
            .compactMap { $0 }
            .map { "\(Format.usd($0.value)) на \(Format.percent($0.offset, digits: 1))" }
            .joined(separator: " · ")
    }
}
