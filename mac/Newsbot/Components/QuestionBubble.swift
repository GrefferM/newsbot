import SwiftUI

struct QuestionBubble: View {
    let text: String

    private static let width: CGFloat = 520

    var body: some View {
        Text(text)
            .font(.system(size: 12))
            .textSelection(.enabled)
            .foregroundStyle(.white)
            .padding(.horizontal, 10)
            .padding(.vertical, 6)
            .background(Theme.accent, in: RoundedRectangle(cornerRadius: 12))
            .frame(maxWidth: Self.width, alignment: .trailing)
            .frame(maxWidth: .infinity, alignment: .trailing)
    }
}
