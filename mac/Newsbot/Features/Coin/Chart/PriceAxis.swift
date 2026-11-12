import Charts
import NewsbotCore
import SwiftUI

enum PriceAxis {
    static let width: CGFloat = 58

    @AxisContentBuilder
    static func marks(aim: Double?, last: Candle?) -> some AxisContent {
        AxisMarks(position: .trailing) { value in
            AxisGridLine()
                .foregroundStyle(Theme.faintRule)

            if let price = value.as(Double.self) {
                AxisValueLabel {
                    Text(Format.price(price))
                        .font(.system(size: 10, design: .monospaced))
                        .foregroundStyle(Theme.caption)
                        .frame(width: Self.width, alignment: .leading)
                }
            }
        }

        AxisMarks(position: .trailing, values: aim.map { [$0] } ?? []) { _ in
            AxisValueLabel(collisionResolution: .disabled) {
                Text(Format.price(aim ?? 0))
                    .font(.system(size: 10, weight: .semibold, design: .monospaced))
                    .foregroundStyle(Color.black.opacity(0.85))
                    .padding(.vertical, 1)
                    .frame(width: Self.width, alignment: .leading)
                    .background(Color(white: 0.88), in: RoundedRectangle(cornerRadius: 3))
            }
        }

        AxisMarks(position: .trailing, values: last.map { [$0.close] } ?? []) { _ in
            AxisValueLabel(collisionResolution: .disabled) {
                if let last {
                    Text(Format.price(last.close))
                        .font(.system(size: 10, weight: .semibold, design: .monospaced))
                        .foregroundStyle(Color.black.opacity(0.85))
                        .padding(.vertical, 1)
                        .frame(width: Self.width, alignment: .leading)
                        .background(last.rising ? Theme.green : Theme.red, in: RoundedRectangle(cornerRadius: 3))
                }
            }
        }
    }
}
