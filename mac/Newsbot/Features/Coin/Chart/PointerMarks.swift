import Charts
import NewsbotCore
import SwiftUI

struct PointerMarks: ChartContent {
    let shown: Candle?
    let aim: Double?
    let last: Candle?

    var body: some ChartContent {
        if let shown {
            RuleMark(x: .value("Свічка", shown.time))
                .lineStyle(StrokeStyle(lineWidth: 1))
                .foregroundStyle(Color.white.opacity(0.25))
                .annotation(
                    position: .bottom, alignment: .center, spacing: 2,
                    overflowResolution: .init(x: .fit(to: .chart), y: .disabled)
                ) {
                    Text(shown.opened())
                        .font(.system(size: 10, weight: .semibold, design: .monospaced))
                        .foregroundStyle(Color.black.opacity(0.85))
                        .padding(.horizontal, 4)
                        .padding(.vertical, 1)
                        .background(Color(white: 0.88), in: RoundedRectangle(cornerRadius: 3))
                }
        }

        if let aim {
            RuleMark(y: .value("Рівень вказівника", aim))
                .lineStyle(StrokeStyle(lineWidth: 1))
                .foregroundStyle(Color.white.opacity(0.25))
        }

        if let last {
            RuleMark(y: .value("Остання ціна", last.close))
                .lineStyle(StrokeStyle(lineWidth: 1, dash: [2, 3]))
                .foregroundStyle((last.rising ? Theme.green : Theme.red).opacity(0.7))
        }
    }
}
