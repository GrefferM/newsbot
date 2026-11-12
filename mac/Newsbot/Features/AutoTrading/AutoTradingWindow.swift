import NewsbotCore
import SwiftUI

struct AutoTradingWindow: View {
    @Environment(AppStore.self) private var store
    @Environment(\.openWindow) private var openWindow

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            header
                .padding(.horizontal, 24)

            ScrollView {
                strategies
                    .padding(.horizontal, 24)
                    .padding(.bottom, 24)
            }
            .scrollIndicators(.automatic)
        }
        .padding(.top, 40)
        .frame(width: 560, alignment: .leading)
        .frame(minHeight: 420, idealHeight: 860, maxHeight: .infinity, alignment: .top)
        .background(Theme.dialog)
        .foregroundStyle(Theme.text)
        .preferredColorScheme(.dark)
    }

    private var strategies: some View {
        VStack(alignment: .leading, spacing: 16) {
            if let auto = store.auto {
                ForEach(auto.strategies) { strategy in
                    StrategyCard(strategy: strategy)
                }
            } else {
                Text("Сервер ще не відповів")
                    .font(.system(size: 12))
                    .foregroundStyle(Theme.secondary)
            }

            if let error = store.actionError {
                Text(error)
                    .font(.system(size: 12))
                    .foregroundStyle(Theme.redText)
            }
        }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack(spacing: 8) {
                Text("Автотрейдинг")
                    .font(.system(size: 20, weight: .bold))

                Pill(
                    text: store.book.venueName.uppercased(), foreground: Theme.yellow,
                    background: Theme.yellow.opacity(0.18))

                Spacer()

                if let auto = store.auto {
                    Text(auto.countLine)
                        .font(.system(size: 12))
                        .foregroundStyle(Theme.secondary)
                }
            }

            HStack(spacing: 8) {
                Text("Кожна стратегія торгує сама за своїми правилами; угоди — у позиціях і стрічці.")
                    .font(.system(size: 12))
                    .foregroundStyle(Theme.secondary)

                Spacer()

                PillButton(icon: "chart.xyaxis.line", title: "Загальний стенд") {
                    openWindow(id: "stand", value: StandWindow.overall)
                }
            }
        }
    }
}
