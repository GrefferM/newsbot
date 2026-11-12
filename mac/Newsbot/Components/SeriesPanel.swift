import Charts
import NewsbotCore
import SwiftUI

struct SeriesPanel: View {
    enum Shape {
        case line
        case bars
    }

    let points: [SeriesPoint]
    let color: Color
    let shape: Shape
    let period: ClosedRange<Date>
    let ticks: [TimeMark]
    let dated: Bool
    let value: (Double) -> String

    @Binding var pointed: Date?

    var body: some View {
        Chart {
            ForEach(points) { point in
                switch shape {
                case .line:
                    LineMark(x: .value("Час", point.time), y: .value("Значення", point.value))
                        .foregroundStyle(color)
                        .lineStyle(StrokeStyle(lineWidth: 1.4))
                case .bars:
                    RectangleMark(
                        x: .value("Час", point.time),
                        yStart: .value("Нуль", 0),
                        yEnd: .value("Значення", point.value),
                        width: .fixed(points.count > 120 ? 1.5 : 3)
                    )
                    .foregroundStyle(point.value < 0 ? Theme.redText : color)
                }
            }

            if shape == .bars {
                RuleMark(y: .value("Нуль", 0))
                    .lineStyle(StrokeStyle(lineWidth: 1))
                    .foregroundStyle(Theme.raised)
            }

            if let at = pointed.flatMap(points.nearest(to:)) {
                RuleMark(x: .value("Момент", at.time))
                    .lineStyle(StrokeStyle(lineWidth: 1))
                    .foregroundStyle(Color.white.opacity(0.25))

                if shape == .line {
                    PointMark(x: .value("Момент", at.time), y: .value("Значення", at.value))
                        .symbolSize(26)
                        .foregroundStyle(color)
                }
            }
        }
        .chartXScale(domain: period)
        .chartYScale(domain: .automatic(includesZero: shape == .bars))
        .chartYAxis {
            AxisMarks(position: .trailing, values: .automatic(desiredCount: 3)) { mark in
                AxisGridLine()
                    .foregroundStyle(Theme.faintRule)

                if let number = mark.as(Double.self) {
                    AxisValueLabel {
                        Text(value(number))
                            .font(.system(size: 10, design: .monospaced))
                            .foregroundStyle(Theme.caption)
                            .frame(width: Self.scaleWidth, alignment: .leading)
                    }
                }
            }
        }
        .chartXAxis {
            AxisMarks(values: ticks.map(\.time)) { mark in
                let tick = ticks.first { $0.time == mark.as(Date.self) }

                AxisGridLine(stroke: StrokeStyle(lineWidth: 1, dash: [2, 4]))
                    .foregroundStyle(Theme.rule)

                if dated {
                    AxisValueLabel(anchor: .top) {
                        Text(tick?.label ?? "")
                            .font(.system(size: 10, design: .monospaced))
                            .foregroundStyle(Theme.caption)
                    }
                }
            }
        }
        .chartOverlay { proxy in
            GeometryReader { area in
                Rectangle()
                    .fill(.clear)
                    .contentShape(Rectangle())
                    .onContinuousHover { phase in
                        point(phase, proxy: proxy, area: area)
                    }
            }
        }
        .frame(height: dated ? 70 : 54)
    }

    private static let scaleWidth: CGFloat = 52

    private func point(_ phase: HoverPhase, proxy: ChartProxy, area: GeometryProxy) {
        guard case .active(let location) = phase, let frame = proxy.plotFrame else {
            pointed = nil

            return
        }

        pointed = proxy.value(atX: location.x - area[frame].origin.x, as: Date.self)
    }
}
