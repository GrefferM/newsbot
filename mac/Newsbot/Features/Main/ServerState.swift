import NewsbotCore
import SwiftUI

struct ServerState: View {
    @Environment(AppStore.self) private var store
    @State private var listed = false

    var body: some View {
        Button {
            listed.toggle()
        } label: {
            HStack(spacing: 6) {
                Circle()
                    .fill(store.online ? Theme.green : Theme.red)
                    .frame(width: 8, height: 8)

                Text(label)
                    .font(.system(size: 12))
                    .foregroundStyle(Theme.secondary)
            }
        }
        .buttonStyle(.plain)
        .help(store.failure ?? "Показати джерела")
        .popover(isPresented: $listed, arrowEdge: .bottom) {
            SourceList(sources: store.health?.sources ?? [:])
        }
    }

    private var label: String {
        if store.online {
            return "Сервер онлайн · джерела \(store.sourcesOK) з \(store.sourcesTotal)"
        }

        return "Сервер недоступний"
    }
}
