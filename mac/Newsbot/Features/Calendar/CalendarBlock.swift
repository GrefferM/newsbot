import NewsbotCore
import SwiftUI

struct CalendarBlock: View {
    let tradfi: Bool

    @Environment(AppStore.self) private var store
    @Environment(Navigator.self) private var navigator
    @Environment(\.openURL) private var openURL
    @Environment(\.openWindow) private var openWindow

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text("Календар")
                    .font(.system(size: 15, weight: .semibold))

                Text(tradfi ? "статистика США, ФРС і звіти компаній" : "розблокування, дати монет, експірації")
                    .font(.system(size: 12))
                    .foregroundStyle(Theme.caption)
            }

            let upcoming = store.calendar(tradfi: tradfi)

            if upcoming.isEmpty {
                Text("Найближчі події з'являться тут, щойно сервер їх збере.")
                    .font(.system(size: 12))
                    .foregroundStyle(Theme.secondary)
            }

            ForEach(upcoming) { event in
                CalendarRow(event: event) {
                    open(event)
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 16)
        .padding(.top, 12)
        .padding(.bottom, 14)
        .overlay(alignment: .top) {
            Theme.rule
                .frame(height: 1)
        }
    }

    private func open(_ event: MarketEvent) {
        if let coin = event.coin {
            navigator.show(coin: coin)
            openWindow(id: "coin")
        } else if let address = event.url.flatMap(URL.init(string:)) {
            openURL(address)
        }
    }
}
