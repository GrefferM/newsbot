import Charts
import NewsbotCore
import SwiftUI

enum PriceAxis {
    static let width: CGFloat = 72

    @AxisContentBuilder
    static func marks(aim: Double?, last: Candle?, prices: ClosedRange<Double>) -> some AxisContent {
        AxisMarks(position: .trailing) { value in
            AxisGridLine()
                .foregroundStyle(Theme.faintRule)

            if let price = value.as(Double.self), !covered(price, by: [aim, last?.close], in: prices) {
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
                plate(Format.price(aim ?? 0), text: Color.black.opacity(0.85), fill: Color(white: 0.88))
            }
        }

        AxisMarks(position: .trailing, values: last.map { [$0.close] } ?? []) { _ in
            AxisValueLabel(collisionResolution: .disabled) {
                if let last {
                    plate(Format.price(last.close), text: .white, fill: last.rising ? Theme.green : Theme.red)
                }
            }
        }
    }

    private static func plate(_ value: String, text: Color, fill: Color) -> some View {
        Text(value)
            .font(.system(size: 11, weight: .bold, design: .monospaced))
            .foregroundStyle(text)
            .padding(.horizontal, 6)
            .padding(.vertical, 3)
            .frame(width: Self.width, alignment: .leading)
            .background(fill, in: RoundedRectangle(cornerRadius: 4))
    }

    private static func covered(_ price: Double, by plates: [Double?], in prices: ClosedRange<Double>) -> Bool {
        let near = (prices.upperBound - prices.lowerBound) * 0.06

        return plates.contains { plate in
            plate.map { abs($0 - price) < near } ?? false
        }
    }
}
