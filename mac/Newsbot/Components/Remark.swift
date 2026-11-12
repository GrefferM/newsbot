import NewsbotCore
import SwiftUI

struct Remark: View {
    let text: String

    var body: some View {
        Text(text)
            .font(.system(size: 12))
            .foregroundStyle(Theme.caption)
            .fixedSize(horizontal: false, vertical: true)
    }
}
