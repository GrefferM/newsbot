import SwiftUI

struct AskField: View {
    let prompt: String
    let title: String
    let busy: Bool
    let send: (String) async -> Void

    @State private var text = ""

    var body: some View {
        HStack(spacing: 8) {
            TextField(prompt, text: $text)
                .textFieldStyle(.plain)
                .font(.system(size: 12))
                .padding(.horizontal, 10)
                .padding(.vertical, 5)
                .background(Theme.field, in: RoundedRectangle(cornerRadius: 7))
                .onSubmit(submit)

            ActionButton(title: title) {
                await take()
            }
            .disabled(busy)
        }
    }

    private func submit() {
        guard !busy else {
            return
        }

        Task {
            await take()
        }
    }

    private func take() async {
        let taken = text

        text = ""

        await send(taken)
    }
}
