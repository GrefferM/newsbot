import NewsbotCore
import SwiftUI

struct TimedLine: View {
    let left: String
    let right: String

    var body: some View {
        HStack(alignment: .firstTextBaseline, spacing: 10) {
            Text(left)
                .font(.system(size: 12, design: .monospaced))
                .foregroundStyle(Theme.secondary)
                .frame(width: 84, alignment: .leading)

            Text(right)
                .font(.system(size: 12))
        }
    }
}
