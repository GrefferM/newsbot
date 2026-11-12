import NewsbotCore
import SwiftUI

enum CoinTab: String, CaseIterable, Sendable {
    case watchlist = "Вішліст"
    case coins = "Монети"
    case tradfi = "TradFi"
}

struct WatchlistColumn: View {
    @Environment(AppStore.self) private var store
    @Environment(\.openWindow) private var openWindow
    @Environment(Navigator.self) private var navigator

    @AppStorage("coinTab") private var tab = CoinTab.coins
    @State private var query = ""

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(spacing: 4) {
                ForEach(CoinTab.allCases, id: \.self) { option in
                    Button {
                        tab = option
                    } label: {
                        Text(option.rawValue)
                            .font(.system(size: 15, weight: tab == option ? .semibold : .regular))
                            .foregroundStyle(tab == option ? Theme.text : Theme.secondary)
                    }
                    .buttonStyle(.plain)
                    .padding(.trailing, 10)
                }

                Text(note)
                    .font(.system(size: 12))
                    .foregroundStyle(Theme.caption)

                Spacer()
            }
            .padding(.horizontal, 16)
            .padding(.top, 14)
            .padding(.bottom, 10)

            SearchField(
                prompt: search, text: $query
            )
            .padding(.horizontal, 16)
            .padding(.bottom, 8)

            switch tab {
            case .coins:
                PulseTable(board: store.pulse, query: query) { asset in
                    navigator.show(coin: asset)
                    openWindow(id: "coin")
                }
            case .tradfi:
                PulseTable(board: store.tradfi, query: query) { asset in
                    navigator.show(coin: asset)
                    openWindow(id: "coin")
                }
            case .watchlist:
                PulseTable(
                    board: store.board(of: searching ? store.search(query) : store.watchlist.coins), query: "",
                    keeping: searching ? .browsing : .kept, empty: empty
                ) { asset in
                    navigator.show(coin: asset)
                    openWindow(id: "coin")
                }
            }

            CalendarBlock(tradfi: tab == .tradfi)
        }
    }

    private var searching: Bool {
        !query.trimmingCharacters(in: .whitespaces).isEmpty
    }

    private var search: String {
        switch tab {
        case .watchlist:
            "Знайти монету за тікером, щоб додати"
        case .coins:
            "Знайти монету"
        case .tradfi:
            "Знайти контракт"
        }
    }

    private var note: String {
        switch tab {
        case .watchlist:
            "\(store.watchlist.coins.count)"
        case .coins:
            note(of: store.pulse, quiet: "тихо")
        case .tradfi:
            note(of: store.tradfi, quiet: "сировина й акції США · тихо")
        }
    }

    private func note(of board: PulseBoard, quiet: String) -> String {
        let trends = board.coins.filter { $0.trend != nil }.count
        let woke = board.movers == 0 ? quiet : "ожили: \(board.movers)"

        return trends == 0 ? woke : "\(woke) · тенденція: \(trends)"
    }

    private var empty: String {
        if searching {
            return store.coins.coins.isEmpty ? "Сервер ще сканує монети." : "Нічого не знайдено."
        }

        return "Вішліст порожній. Введи тікер у пошук і натисни «+» або додай монету з її сторінки."
    }
}
