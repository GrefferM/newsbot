import NewsbotCore
import SwiftUI

struct AutoTradingButton: View {
    @Environment(AppStore.self) private var store
    @Environment(\.openWindow) private var openWindow

    var body: some View {
        Button {
            openWindow(id: "auto")
        } label: {
            HStack(spacing: 6) {
                Circle()
                    .fill(running ? Theme.green : Theme.secondary.opacity(0.5))
                    .frame(width: 7, height: 7)

                Text("Автотрейдинг")
                    .font(.system(size: 12))
                    .foregroundStyle(running ? Theme.text : Theme.secondary)
            }
        }
        .buttonStyle(.plain)
        .help("Стратегії автотрейдингу та їхні тумблери")
    }

    private var running: Bool {
        store.auto?.running == true
    }
}
