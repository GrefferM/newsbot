import NewsbotCore
import SwiftUI

let tableColumns = [
    GridItem(.flexible(minimum: 84), spacing: 4, alignment: .leading),
    GridItem(.fixed(80), spacing: 4, alignment: .trailing),
    GridItem(.fixed(50), spacing: 4, alignment: .trailing),
    GridItem(.fixed(50), spacing: 4, alignment: .trailing),
    GridItem(.fixed(50), spacing: 4, alignment: .trailing),
    GridItem(.fixed(58), spacing: 4, alignment: .trailing),
    GridItem(.fixed(44), spacing: 4, alignment: .trailing),
]

enum Keeping {
    case none
    case kept
    case browsing
}

struct PulseTable: View {
    let board: PulseBoard
    let query: String
    var keeping = Keeping.none
    var empty = "Тут такого немає."
    let open: (String) -> Void

    @State private var sort = PulseSort.activity
    @State private var ascending = false

    var body: some View {
        LazyVGrid(columns: tableColumns, spacing: 0) {
            SortTitle(title: "Монета", key: PulseSort.asset, rising: true, sort: $sort, ascending: $ascending)
            Text("Ціна")
            SortTitle(title: "15хв", key: PulseSort.quarter, sort: $sort, ascending: $ascending)
            SortTitle(title: "1год", key: PulseSort.hour, sort: $sort, ascending: $ascending)
            SortTitle(title: "4год", key: PulseSort.fourHours, sort: $sort, ascending: $ascending)
            SortTitle(title: "Обіг", key: PulseSort.surge, sort: $sort, ascending: $ascending)
            SortTitle(title: "Куп.", key: PulseSort.buyers, sort: $sort, ascending: $ascending)
        }
        .font(.system(size: 11))
        .foregroundStyle(Theme.caption)
        .padding(.horizontal, 12)
        .padding(.vertical, 6)
        .overlay(alignment: .bottom) {
            Theme.rule
                .frame(height: 1)
        }

        ScrollView {
            LazyVStack(spacing: 0) {
                if rows.isEmpty {
                    Text(board.coins.isEmpty && keeping == .none ? "Сервер ще читає ринок." : empty)
                        .font(.system(size: 13))
                        .foregroundStyle(Theme.secondary)
                        .padding(16)
                }

                ForEach(rows) { coin in
                    PulseRow(coin: coin, keeping: keeping)
                        .contentShape(Rectangle())
                        .onTapGesture {
                            open(coin.asset)
                        }
                }
            }
        }
    }
}

extension PulseTable {
    fileprivate var rows: [PulseCoin] {
        sort.sorted(board.found(query), ascending: ascending)
    }
}
