import Charts
import NewsbotCore
import SwiftUI

struct EventMarks: ChartContent {
    let marks: [ChartMark]
    let shown: Date?

    var body: some ChartContent {
        ForEach(marks) { mark in
            let told = mark.ahead || mark.time == shown

            RuleMark(x: .value("Подія", mark.time))
                .lineStyle(StrokeStyle(lineWidth: 1, dash: mark.ahead ? [4, 3] : []))
                .foregroundStyle(Theme.yellow.opacity(told ? 0.9 : 0.3))
                .annotation(
                    position: .top, alignment: mark.ahead ? .trailing : .center, spacing: 2,
                    overflowResolution: .init(x: .fit(to: .chart), y: .disabled)
                ) {
                    EventFlag(text: told ? mark.label : nil)
                }
        }
    }
}
