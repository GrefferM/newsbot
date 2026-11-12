import Charts
import NewsbotCore
import SwiftUI

struct DepthChart: View {
    let curve: [DepthPoint]

    var body: some View {
        Chart(curve) { point in
            AreaMark(
                x: .value("Відстань", point.offset),
                y: .value("Глибина", point.total),
                series: .value("Бік", point.bid ? "bid" : "ask")
            )
            .interpolationMethod(point.bid ? .stepEnd : .stepStart)
            .foregroundStyle((point.bid ? Theme.green : Theme.red).opacity(0.22))

            LineMark(
                x: .value("Відстань", point.offset),
                y: .value("Глибина", point.total),
                series: .value("Бік", point.bid ? "bid" : "ask")
            )
            .interpolationMethod(point.bid ? .stepEnd : .stepStart)
            .lineStyle(StrokeStyle(lineWidth: 1.4))
            .foregroundStyle(point.bid ? Theme.green : Theme.red)
        }
        .chartXScale(domain: -Liquidity.far...Liquidity.far)
        .chartXAxis {
            AxisMarks(values: [-0.02, -0.01, 0, 0.01, 0.02]) { mark in
                let offset = mark.as(Double.self) ?? 0

                AxisGridLine(stroke: StrokeStyle(lineWidth: 1, dash: offset == 0 ? [] : [2, 4]))
                    .foregroundStyle(offset == 0 ? Theme.raised : Theme.rule)

                AxisValueLabel(anchor: .top) {
                    Text(offset == 0 ? "ціна" : Format.percent(offset))
                        .font(.system(size: 10, design: .monospaced))
                        .foregroundStyle(Theme.caption)
                }
            }
        }
        .chartYAxis {
            AxisMarks(position: .trailing, values: .automatic(desiredCount: 3)) { mark in
                AxisGridLine()
                    .foregroundStyle(Theme.faintRule)

                if let total = mark.as(Double.self) {
                    AxisValueLabel {
                        Text(Format.usd(total))
                            .font(.system(size: 10, design: .monospaced))
                            .foregroundStyle(Theme.caption)
                    }
                }
            }
        }
        .frame(maxHeight: .infinity)
    }
}
