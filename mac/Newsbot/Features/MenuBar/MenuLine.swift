import NewsbotCore
import SwiftUI

struct MenuLine: View {
    let name: String
    let value: String
    let tint: Color

    var body: some View {
        HStack {
            Text(name)

            Spacer()

            Text(value)
                .font(.system(size: 13, design: .monospaced))
                .foregroundStyle(tint)
        }
        .font(.system(size: 13))
        .padding(.horizontal, 4)
        .padding(.vertical, 6)
    }
}
