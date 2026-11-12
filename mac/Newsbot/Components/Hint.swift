import NewsbotCore
import SwiftUI

struct Hint: View {
    let title: String
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            Text(title)
                .font(.system(size: 12))
                .lineLimit(1)
                .padding(.horizontal, 10)
                .padding(.vertical, 4)
                .foregroundStyle(Theme.text.opacity(0.8))
                .overlay(Capsule().stroke(Color.white.opacity(0.14)))
        }
        .buttonStyle(.plain)
    }
}
