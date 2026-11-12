import Charts
import SwiftUI

struct LevelMarks: ChartContent {
    let levels: [PriceLevel]

    var body: some ChartContent {
        ForEach(levels) { level in
            RuleMark(y: .value(level.label, level.price))
                .lineStyle(StrokeStyle(lineWidth: 1))
                .foregroundStyle(level.color.opacity(0.8))
                .annotation(position: .top, alignment: .trailing, spacing: 1) {
                    Text(level.label)
                        .font(.system(size: 10))
                        .foregroundStyle(level.color)
                }
        }
    }
}
