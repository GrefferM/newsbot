import SwiftUI

struct ActionButton: View {
    let title: String
    let action: () async -> Void

    var body: some View {
        Button(title) {
            Task {
                await action()
            }
        }
        .buttonStyle(.plain)
        .font(.system(size: 12, weight: .semibold))
        .padding(.horizontal, 12)
        .padding(.vertical, 5)
        .background(Theme.control, in: RoundedRectangle(cornerRadius: 7))
    }
}
