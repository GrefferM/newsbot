import NewsbotCore
import SwiftUI

struct MainWindow: View {
    @Environment(AppStore.self) private var store

    var body: some View {
        VStack(spacing: 0) {
            HStack(spacing: 0) {
                WatchlistColumn()
                    .frame(maxWidth: .infinity)
                    .background(Theme.sidebar)

                Divider().overlay(Theme.rule)

                EventsColumn()
                    .frame(maxWidth: .infinity)

                Divider().overlay(Theme.rule)

                AgentsColumn()
                    .frame(maxWidth: .infinity)
            }
        }
        .background(Theme.window)
        .foregroundStyle(Theme.text)
        .frame(minWidth: 1340, minHeight: 640)
        .preferredColorScheme(.dark)
        .toolbar {
            PlainToolbarItem(placement: .navigation) {
                HStack(spacing: 16) {
                    Text("Newsbot")
                        .font(.system(size: 13, weight: .semibold))

                    ServerState()
                }
                .padding(.horizontal, 8)
            }

            ToolbarGap()

            PlainToolbarItem(placement: .primaryAction) {
                HStack(spacing: 16) {
                    if let error = store.actionError {
                        Text(error)
                            .font(.system(size: 12))
                            .foregroundStyle(Theme.redText)
                            .lineLimit(1)
                    }

                    AutoTradingToggle()
                    ResultsButton()
                }
                .padding(.horizontal, 8)
            }
        }
        .toolbarBackground(Theme.titleBar, for: .windowToolbar)
    }
}
