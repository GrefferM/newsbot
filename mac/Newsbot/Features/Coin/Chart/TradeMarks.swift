import Charts
import NewsbotCore
import SwiftUI

struct TradeMarks: ChartContent {
    let trades: [ChartTrade]
    let shown: Date?

    var body: some ChartContent {
        ForEach(trades) { trade in
            let color = trade.side == .short ? Theme.redText : Theme.greenText

            LineMark(
                x: .value("Час", trade.entry),
                y: .value("Ціна", trade.entryPrice),
                series: .value("Угода", trade.id)
            )
            .lineStyle(StrokeStyle(lineWidth: 1, dash: [4, 4]))
            .foregroundStyle(Theme.text.opacity(0.55))

            LineMark(
                x: .value("Час", trade.exit),
                y: .value("Ціна", trade.exitPrice),
                series: .value("Угода", trade.id)
            )
            .lineStyle(StrokeStyle(lineWidth: 1, dash: [4, 4]))
            .foregroundStyle(Theme.text.opacity(0.55))

            PointMark(x: .value("Вхід", trade.entry), y: .value("Ціна входу", trade.entryPrice))
                .symbol {
                    Image(systemName: trade.side == .short ? "arrowtriangle.down.fill" : "arrowtriangle.up.fill")
                        .font(.system(size: 11))
                        .foregroundStyle(color)
                        .offset(y: trade.side == .short ? -10 : 10)
                }

            PointMark(x: .value("Вихід", trade.exit), y: .value("Ціна виходу", trade.exitPrice))
                .symbol {
                    Circle()
                        .strokeBorder(Theme.text, lineWidth: 2)
                        .background(Circle().fill(Theme.card))
                        .frame(width: 11, height: 11)
                }
                .annotation(
                    position: .top, alignment: .center, spacing: 8,
                    overflowResolution: .init(x: .fit(to: .chart), y: .fit(to: .chart))
                ) {
                    if trade.touches(shown) {
                        TradeNote(trade: trade)
                    }
                }
        }
    }
}

struct TradeNote: View {
    let trade: ChartTrade

    var body: some View {
        VStack(alignment: .leading, spacing: 3) {
            Text(trade.lines[0])
                .fontWeight(.semibold)

            Text(trade.lines[1])
                .foregroundStyle(Theme.secondary)

            Text(trade.lines[2])
                .fontWeight(.semibold)
                .foregroundStyle(trade.won ? Theme.greenText : Theme.redText)
        }
        .font(.system(size: 11, design: .monospaced))
        .padding(.horizontal, 10)
        .padding(.vertical, 7)
        .background(Theme.raised, in: RoundedRectangle(cornerRadius: 8))
    }
}
