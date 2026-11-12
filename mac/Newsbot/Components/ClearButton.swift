import SwiftUI

struct ClearButton: View {
    let help: String
    let action: () async -> Void

    var body: some View {
        Button("Очистити історію") {
            Task {
                await action()
            }
        }
        .buttonStyle(.plain)
        .font(.system(size: 11))
        .foregroundStyle(Theme.secondary)
        .help(help)
    }
}
