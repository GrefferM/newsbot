import Charts
import NewsbotCore
import SwiftUI

struct Tile: View {
    let label: String
    let value: String
    let note: String
    let tint: Color

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(label)
                .font(.system(size: 11))
                .foregroundStyle(Theme.secondary)

            Text(value)
                .font(.system(size: 18, weight: .semibold, design: .monospaced))
                .foregroundStyle(tint)

            Text(note)
                .font(.system(size: 11))
                .foregroundStyle(Theme.caption)
                .lineLimit(1)
                .minimumScaleFactor(0.8)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 14)
        .padding(.vertical, 12)
        .background(Theme.card, in: RoundedRectangle(cornerRadius: 10))
    }
}
