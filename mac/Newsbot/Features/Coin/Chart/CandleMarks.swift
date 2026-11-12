import Charts
import NewsbotCore
import SwiftUI

struct CandleMarks: ChartContent {
    let candles: [Candle]

    var body: some ChartContent {
        ForEach(candles) { candle in
            RuleMark(
                x: .value("Час", candle.time),
                yStart: .value("Мінімум", candle.low),
                yEnd: .value("Максимум", candle.high)
            )
            .lineStyle(StrokeStyle(lineWidth: 1))
            .foregroundStyle(candle.rising ? Theme.green : Theme.red)

            RectangleMark(
                x: .value("Час", candle.time),
                yStart: .value("Відкриття", candle.open),
                yEnd: .value("Закриття", candle.close),
                width: .fixed(4)
            )
            .foregroundStyle(candle.rising ? Theme.green : Theme.red)
        }
    }
}
