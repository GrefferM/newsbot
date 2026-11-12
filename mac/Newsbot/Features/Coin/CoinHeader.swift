import NewsbotCore
import SwiftUI

struct CoinHeader: View {
    let asset: String
    let coin: WatchCoin?
    let live: PulseCoin?
    let chart: ChartModel

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack {
                if coin?.tagged == true {
                    Pill(text: "Monitoring Tag", foreground: Theme.redText, background: Theme.red.opacity(0.16))
                }

                if let stage = live?.stage {
                    Pill(text: stage, foreground: Theme.orangeText, background: Theme.orange.opacity(0.16))
                }

                if let trend = live?.trend {
                    Pill(text: trend, foreground: Theme.text, background: Theme.raised)
                }
            }
            .frame(height: 28)
            .padding(.leading, 56)

            HStack(alignment: .center, spacing: 16) {
                Text(asset)
                    .font(.system(size: 28, weight: .bold))

                HStack(spacing: 5) {
                    Circle()
                        .fill(chart.live ? Theme.green : Color.white.opacity(0.25))
                        .frame(width: 7, height: 7)

                    Text(chart.live ? "наживо" : "з'єднання…")
                        .font(.system(size: 12))
                        .foregroundStyle(Theme.secondary)
                }

                Spacer()

                KeepButton(asset: asset)

                if let change = chart.day?.change {
                    Text("\(Format.percent(change, digits: 1)) за добу")
                        .font(.system(size: 13, design: .monospaced))
                        .foregroundStyle(change < 0 ? Theme.redText : Theme.greenText)
                }

                if let price = chart.price {
                    Text(Format.price(price))
                        .font(.system(size: 26, weight: .bold, design: .monospaced))
                        .contentTransition(.numericText())
                }
            }
        }
        .padding(.horizontal, 28)
        .padding(.top, 2)
        .padding(.bottom, 14)
        .background(Theme.titleBar)
    }
}
