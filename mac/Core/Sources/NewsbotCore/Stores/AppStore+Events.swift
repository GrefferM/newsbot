import Foundation

extension AppStore {
    nonisolated static func newestFirst(_ events: [BotEvent]) -> [BotEvent] {
        events.enumerated()
            .sorted { ($0.element.time, $1.offset) > ($1.element.time, $0.offset) }
            .map(\.element)
    }

    nonisolated static func unique(_ events: [BotEvent]) -> [BotEvent] {
        var seen = Set<String>()

        return events.filter { seen.insert($0.id).inserted }
    }

    public func event(_ id: String) -> BotEvent? {
        events.first { $0.id == id }
    }

    public func position(_ id: String) -> Position? {
        book.positions.first { $0.id == id }
    }

    public func filtered(_ filter: EventFilter) -> [BotEvent] {
        switch filter {
        case .all:
            events
        case .orders:
            events.filter { $0.category == .trade }
        case .triggers:
            []
        case .watchlist:
            events.filter { event in
                event.symbols?.contains { symbol in watchlist.coins.contains { $0.asset == symbol } } ?? false
            }
        }
    }
}
