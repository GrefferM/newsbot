import NewsbotCore
import SwiftUI

struct CoinTiles: View {
    let coin: WatchCoin?
    let live: PulseCoin?

    var body: some View {
        HStack(spacing: 12) {
            Tile(
                label: "Обіг за добу", value: (coin?.volumeDay ?? live?.volumeDay).map { Format.usd($0) } ?? "—",
                note: volumeNote, tint: (volumeRatio ?? 0) >= 2 ? Theme.orangeText : Theme.text)
            Tile(
                label: "90 д до BTC",
                value: coin?.trendVsBTC.map { Format.percent($0) } ?? "—",
                note: "зміна відносно BTC",
                tint: (coin?.trendVsBTC ?? 0) < 0 ? Theme.redText : Theme.greenText
            )
            Tile(
                label: "Фандинг / 8 год",
                value: funding.map(Format.funding) ?? "—",
                note: (funding ?? 0) < 0 ? "шорти платять лонгам" : "лонги платять шортам",
                tint: Theme.orangeText
            )
            Tile(
                label: "Відкритий інтерес",
                value: coin?.openInterest.map { Format.usd($0) } ?? "—",
                note: "перп",
                tint: Theme.text
            )
        }
    }

    private var volumeRatio: Double? {
        guard let coin, let day = coin.volumeDay, coin.volume > 0 else {
            return nil
        }

        return day / coin.volume
    }

    private var volumeNote: String {
        if coin?.volumeDay == nil, live?.volumeDay != nil {
            return "перп"
        }

        return volumeRatio.map { String(format: "×%.1f від середнього за 30 д", $0) } ?? "перп"
    }

    private var funding: Double? {
        coin?.funding ?? live?.funding
    }
}
