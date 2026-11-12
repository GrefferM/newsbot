import Charts
import NewsbotCore
import SwiftUI

struct VolumeChart: View {
    let candles: [Candle]
    let domain: ClosedRange<Date>

    var body: some View {
        Chart(candles) { candle in
            RectangleMark(
                x: .value("Час", candle.time),
                yStart: .value("Нуль", 0),
                yEnd: .value("Обсяг", candle.volume),
                width: .fixed(4)
            )
            .foregroundStyle(Color.white.opacity(0.22))
        }
        .chartXScale(domain: domain)
        .chartYAxis {
            AxisMarks(position: .trailing, values: [0]) { _ in
                AxisValueLabel {
                    Color.clear
                        .frame(width: PriceAxis.width, height: 1)
                }
            }
        }
        .chartXAxis(.hidden)
        .frame(height: 36)
    }
}
