import Foundation
import Observation

public enum EventFilter: String, CaseIterable, Sendable {
    case all = "Усі"
    case orders = "Ордери"
    case watchlist = "Вішліст"
    case triggers = "Тригери"
}

@MainActor
@Observable
public final class AppStore {

    public private(set) var watchlist = Watchlist.empty
    public private(set) var coins = Watchlist.empty
    public private(set) var events: [BotEvent] = []
    public private(set) var triggers: [PlanWatch] = []
    public private(set) var results: DeskResults?
    public private(set) var auto: Bool?
    public private(set) var pulse = PulseBoard.empty
    public private(set) var tradfi = PulseBoard.empty
    private var pulsed: [String: PulseCoin] = [:]
    private var scanned: [String: WatchCoin] = [:]
    public private(set) var calendar: [MarketEvent] = []
    public private(set) var book = Book.empty
    public private(set) var health: Health?
    public private(set) var failure: String?
    public private(set) var refreshed: Date?
    public private(set) var actionError: String?

    private var api: BotAPI
    private let eventLimit: Int
    private let pulseEvery: TimeInterval
    private let slowEvery: TimeInterval
    private var pulseAt: Date?
    private var slowAt: Date?

    public init(api: BotAPI, eventLimit: Int = 100, pulseEvery: TimeInterval = 30, slowEvery: TimeInterval = 300) {
        self.api = api
        self.eventLimit = eventLimit
        self.pulseEvery = pulseEvery
        self.slowEvery = slowEvery
    }

    public func connect(to api: BotAPI) {
        self.api = api
        failure = nil
        health = nil
        pulseAt = nil
        slowAt = nil
        connections += 1
    }

    public private(set) var connections = 0

    public var server: BotAPI {
        api
    }

    public var online: Bool {
        failure == nil && health != nil
    }

    public var sourcesOK: Int {
        health?.sources.values.filter(\.working).count ?? 0
    }

    public var sourcesTotal: Int {
        health?.sources.count ?? 0
    }

    public func refresh() async {
        do {
            async let watchlist = api.watchlist()
            async let events = api.events(limit: eventLimit)
            async let book = api.book()
            async let health = api.health()
            async let results = try? api.results()
            async let triggers = try? api.triggers()
            async let auto = try? api.auto()

            let fresh = try await (watchlist: watchlist, events: events, book: book, health: health)

            keep(\.watchlist, fresh.watchlist)
            index()
            keep(\.events, Self.newestFirst(Self.unique(fresh.events).filter { $0.category != .reminder }))
            keep(\.book, fresh.book)
            keep(\.health, fresh.health)
            keep(\.results, await results)
            keep(\.auto, await auto?.enabled)

            if let watched = await triggers {
                keep(\.triggers, watched.sorted { $0.armed > $1.armed })
            }

            keep(\.failure, nil)
            refreshed = Date()

            await refreshPulse()
            await refreshSlow()
        } catch {
            failure = String(describing: error)
        }
    }

    private func refreshPulse() async {
        if let pulseAt, Date().timeIntervalSince(pulseAt) < pulseEvery {
            return
        }

        async let coins = try? api.pulse()
        async let commodities = try? api.tradfi()

        if let board = await coins {
            keep(\.pulse, board)
        }

        if let board = await commodities {
            keep(\.tradfi, board)
        }

        keep(\.pulsed, Dictionary((pulse.coins + tradfi.coins).map { ($0.asset, $0) }) { first, _ in first })

        pulseAt = Date()
    }

    private func refreshSlow() async {
        if let slowAt, Date().timeIntervalSince(slowAt) < slowEvery {
            return
        }

        if let all = try? await api.coins() {
            keep(\.coins, all)
            index()
        }

        await reloadCalendar()

        slowAt = Date()
    }

    private func keep<Value: Equatable>(_ path: ReferenceWritableKeyPath<AppStore, Value>, _ value: Value) {
        if self[keyPath: path] != value {
            self[keyPath: path] = value
        }
    }

    public func reloadCalendar() async {
        if let upcoming = try? await api.calendar() {
            keep(\.calendar, upcoming)
        }
    }

    public func calendar(tradfi wanted: Bool, now: Date = Date()) -> [MarketEvent] {
        let contracts = Set(tradfi.coins.map(\.asset))
        let ahead = calendar.filter { $0.approximate != true && $0.ahead(of: now) }

        return Array(ahead.filter { $0.traded(among: contracts) == wanted }.prefix(6))
    }

    public func traded(_ asset: String) -> Bool {
        tradfi.coins.contains { $0.asset == asset }
    }

    public func dates(of asset: String, now: Date = Date()) -> [MarketEvent] {
        let traded = traded(asset)

        return Array(calendar.filter { $0.concerns(asset, traded: traded) && $0.ahead(of: now) }.prefix(8))
    }

    public func marks(of asset: String) -> [MarketEvent] {
        let traded = traded(asset)

        return calendar.filter { $0.approximate != true && $0.concerns(asset, traded: traded) }
    }

    nonisolated static func newestFirst(_ events: [BotEvent]) -> [BotEvent] {
        events.enumerated()
            .sorted { ($0.element.time, $1.offset) > ($1.element.time, $0.offset) }
            .map(\.element)
    }

    nonisolated static func unique(_ events: [BotEvent]) -> [BotEvent] {
        var seen = Set<String>()

        return events.filter { seen.insert($0.id).inserted }
    }

    public func run(every interval: Duration) async {
        while !Task.isCancelled {
            await refresh()

            try? await Task.sleep(for: interval)
        }
    }

    public func keep(_ asset: String, note: String) async {
        await act { try await self.api.keep(asset: asset, note: note) }
    }

    public func setAuto(_ on: Bool) async {
        await act { try await self.api.setAuto(AutoTrading(enabled: on)) }
    }

    public func unwatch(_ trigger: PlanWatch) async {
        await act { try await self.api.unwatch(trigger.id) }
    }

    public func drop(_ asset: String) async {
        await act { try await self.api.drop(asset: asset) }
    }

    public func search(_ text: String) -> [WatchCoin] {
        let query = text.trimmingCharacters(in: .whitespaces).uppercased()

        guard !query.isEmpty else {
            return coins.coins
        }

        return coins.coins
            .filter { $0.asset.contains(query) }
            .sorted { ($0.asset.hasPrefix(query) ? 0 : 1, $0.asset) < ($1.asset.hasPrefix(query) ? 0 : 1, $1.asset) }
    }

    public func keeps(_ asset: String) -> Bool {
        watchlist.coins.contains { $0.asset == asset }
    }

    public func board(of coins: [WatchCoin]) -> PulseBoard {
        PulseBoard(
            at: pulse.at,
            coins: coins.map { coin in
                pulse(of: coin.asset) ?? PulseCoin(asset: coin.asset, price: coin.price ?? 0)
            }
        )
    }

    public func pulse(of asset: String) -> PulseCoin? {
        pulsed[asset]
    }

    public func coin(_ asset: String) -> WatchCoin? {
        scanned[asset]
    }

    private func index() {
        keep(\.scanned, Dictionary((watchlist.coins + coins.coins).map { ($0.asset, $0) }) { first, _ in first })
    }

    public func forget(_ event: BotEvent) async {
        await act { try await self.api.forget(event: event.id) }
    }

    public func close(_ position: Position) async {
        await act { try await self.api.close(position: position.id) }
    }

    private func act(_ action: () async throws -> ActionResult) async {
        do {
            let result = try await action()
            actionError = result.error
        } catch {
            actionError = String(describing: error)
        }

        await refresh()
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
