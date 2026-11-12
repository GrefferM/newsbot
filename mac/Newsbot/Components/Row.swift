import Charts
import NewsbotCore
import SwiftUI

struct Row: View {
    let name: String
    let value: String
    let tint: Color

    var body: some View {
        HStack {
            Text(name)
                .foregroundStyle(Theme.text.opacity(0.65))

            Spacer()

            Text(value)
                .font(.system(size: 12, design: .monospaced))
                .foregroundStyle(tint)
        }
        .font(.system(size: 12))
    }
}
