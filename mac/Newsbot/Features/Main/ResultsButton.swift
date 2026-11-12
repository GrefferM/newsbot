import NewsbotCore
import SwiftUI

struct ResultsButton: View {
    @Environment(AppStore.self) private var store
    @State private var shown = false

    var body: some View {
        Button {
            shown.toggle()
        } label: {
            Text("Результати")
                .font(.system(size: 12))
                .foregroundStyle(Theme.secondary)
        }
        .buttonStyle(.plain)
        .help("Що стало з планами на стеженні та угодами")
        .popover(isPresented: $shown, arrowEdge: .bottom) {
            VStack(alignment: .leading, spacing: 6) {
                Text("Результати")
                    .font(.system(size: 15, weight: .semibold))

                Text(results.planLine)
                Text(results.tradeLine)
            }
            .font(.system(size: 12))
            .foregroundStyle(Theme.secondary)
            .fixedSize(horizontal: false, vertical: true)
            .padding(18)
            .frame(width: 380, alignment: .leading)
            .background(Theme.window)
            .preferredColorScheme(.dark)
        }
    }

    private var results: DeskResults {
        store.results ?? DeskResults()
    }
}
