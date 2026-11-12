import NewsbotCore
import SwiftUI

struct BalanceBar: View {
    let label: String
    let bid: Double
    let ask: Double

    var body: some View {
        HStack(spacing: 8) {
            Text(Format.usd(bid))
                .font(.system(size: 11, design: .monospaced))
                .foregroundStyle(Theme.greenText)
                .frame(width: 56, alignment: .leading)

            GeometryReader { geometry in
                HStack(spacing: 2) {
                    Capsule()
                        .fill(Theme.green)
                        .frame(width: max(2, (geometry.size.width - 2) * share))

                    Capsule()
                        .fill(Theme.red)
                }
            }
            .frame(height: 8)
            .overlay {
                Text(label)
                    .font(.system(size: 10, weight: .semibold))
                    .foregroundStyle(Color.black.opacity(0.8))
            }

            Text(Format.usd(ask))
                .font(.system(size: 11, design: .monospaced))
                .foregroundStyle(Theme.redText)
                .frame(width: 56, alignment: .trailing)
        }
    }

    private var share: Double {
        bid + ask > 0 ? bid / (bid + ask) : 0.5
    }
}
