import NewsbotCore
import SwiftUI

struct PositionValue: View {
    let text: String

    init(_ text: String) {
        self.text = text
    }

    var body: some View {
        Text(text)
            .font(.system(size: 13, design: .monospaced))
    }
}
