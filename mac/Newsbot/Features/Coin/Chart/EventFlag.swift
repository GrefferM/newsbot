import SwiftUI

struct EventFlag: View {
    let text: String?

    var body: some View {
        if let text {
            Text(text)
                .font(.system(size: 10, weight: .semibold))
                .foregroundStyle(Color.black.opacity(0.85))
                .lineLimit(1)
                .padding(.horizontal, 5)
                .padding(.vertical, 1)
                .background(Theme.yellow, in: RoundedRectangle(cornerRadius: 3))
                .frame(maxWidth: 520)
        } else {
            Circle()
                .fill(Theme.yellow)
                .frame(width: 5, height: 5)
        }
    }
}
