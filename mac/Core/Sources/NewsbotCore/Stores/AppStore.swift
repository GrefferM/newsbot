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
    public private(set) var auto: AutoTrading?
    public private(set) var pulse = PulseBoard.empty
    public private(set) var tradfi = PulseBoard.empty
    var pulsed: [String: PulseCoin] = [:]
    var scanned: [String: WatchCoin] = [:]
    public private(set) var calendar: [MarketEvent] = []
    public private(set) var book = Book.empty
    public private(set) var health: Health?
    public private(set) var failure: String?
    public private(set) var refreshed: Date?
    public internal(set) var actionError: String?
    public private(set) var connections = 0

    var api: BotAPI
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
            keep(\.auto, await auto)

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

    public func reloadCalendar() async {
        if let upcoming = try? await api.calendar() {
            keep(\.calendar, upcoming)
        }
    }

    func keep<Value: Equatable>(_ path: ReferenceWritableKeyPath<AppStore, Value>, _ value: Value) {
        if self[keyPath: path] != value {
            self[keyPath: path] = value
        }
    }

    public func run(every interval: Duration) async {
        while !Task.isCancelled {
            await refresh()

            try? await Task.sleep(for: interval)
        }
    }
}
