import NewsbotCore
import SwiftUI

struct PlanField: View {
    let title: String
    @Binding var text: String

    var body: some View {
        VStack(alignment: .leading, spacing: 3) {
            Text(title)
                .font(.system(size: 11))
                .foregroundStyle(Theme.caption)
                .lineLimit(1)

            TextField(title, text: $text)
                .textFieldStyle(.plain)
                .font(.system(size: 12, design: .monospaced))
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(Theme.field, in: RoundedRectangle(cornerRadius: 6))
        }
    }
}
