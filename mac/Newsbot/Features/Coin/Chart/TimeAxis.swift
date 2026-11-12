import Charts
import NewsbotCore
import SwiftUI

enum TimeAxis {
    @AxisContentBuilder
    static func marks(_ ticks: [TimeMark]) -> some AxisContent {
        AxisMarks(values: ticks.map(\.time)) { value in
            let tick = ticks.first { $0.time == value.as(Date.self) }
            let major = tick?.major == true

            AxisGridLine(stroke: StrokeStyle(lineWidth: 1, dash: major ? [] : [2, 4]))
                .foregroundStyle(major ? Theme.raised : Theme.faintRule)

            AxisValueLabel(anchor: .top) {
                Text(tick?.label ?? "")
                    .font(.system(size: 10, weight: major ? .semibold : .regular, design: .monospaced))
                    .foregroundStyle(major ? Theme.text : Theme.caption)
            }
        }
    }
}
