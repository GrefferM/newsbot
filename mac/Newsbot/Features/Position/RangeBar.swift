import NewsbotCore
import SwiftUI

struct RangeBar: View {
    let position: Position
    let price: Double?

    var body: some View {
        VStack(spacing: 4) {
            GeometryReader { geometry in
                let width = geometry.size.width

                ZStack(alignment: .leading) {
                    LinearGradient(
                        colors: long ? colors.reversed() : colors,
                        startPoint: .leading,
                        endPoint: .trailing
                    )
                    .frame(height: 6)
                    .clipShape(Capsule())

                    Rectangle()
                        .fill(Color.white.opacity(0.7))
                        .frame(width: 2, height: 18)
                        .offset(x: width * share(position.entry) - 1)

                    if let price {
                        Circle()
                            .fill(.white)
                            .frame(width: 14, height: 14)
                            .overlay(Circle().stroke(Theme.window, lineWidth: 3))
                            .offset(x: width * share(price) - 7)
                    }
                }
                .frame(height: 20)
            }
            .frame(height: 20)

            HStack {
                Text(long ? "SL \(Format.price(position.stop))" : "TP \(Format.price(position.target))")
                Spacer()
                Text("вхід \(Format.price(position.entry))")
                Spacer()
                Text(long ? "TP \(Format.price(position.target))" : "SL \(Format.price(position.stop))")
            }
            .font(.system(size: 11, design: .monospaced))
            .foregroundStyle(Theme.secondary)
        }
    }

    private var long: Bool {
        position.side == .long
    }

    private var colors: [Color] {
        [Theme.green.opacity(0.5), Theme.raised, Theme.red.opacity(0.5)]
    }

    private func share(_ value: Double) -> Double {
        let low = min(position.stop, position.target)
        let span = max(position.stop, position.target) - low

        guard span > 0 else {
            return 0.5
        }

        return min(1, max(0, (value - low) / span))
    }
}
