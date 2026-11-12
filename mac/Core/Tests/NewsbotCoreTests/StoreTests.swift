import Foundation
import Testing

@testable import NewsbotCore

@Test func decodesTheWatchlistAndTheBook() throws {
    let watchlist = """
        {"built": "2026-11-09T18:04:41.891707Z", "coins": [
          {"asset": "MOVE", "reasons": ["monitoring-tag", "low-volume"], "tagged": true, "volume_30d": 219241,
           "trend_90d_vs_btc": -0.29, "contract": "MOVEUSDT", "funding": 0.00005, "open_interest": 1138623},
          {"asset": "QKC", "reasons": ["monitoring-tag"], "tagged": true, "volume_30d": 400000, "trend_90d_vs_btc": null}
        ]}
        """
    let book = """
        {"venue": "testnet", "start": 1000, "equity": 1024.8, "exposure": 0.2, "positions": [
          {"id": "MOVEUSDT-1", "symbol": "MOVE", "contract": "MOVEUSDT", "side": "short", "kind": "Повний делістинг Binance",
           "size": 0.2, "stake": 200, "entry": 1, "stop": 1.3, "target": 0.7,
           "opened": "2026-11-09T11:30:02Z", "deadline": "2026-11-10T11:30:02Z", "open": true}
        ]}
        """

    let list = try ServerJSON.decoder.decode(Watchlist.self, from: Data(watchlist.utf8))
    let account = try ServerJSON.decoder.decode(Book.self, from: Data(book.utf8))

    #expect(list.coins.map(\.asset) == ["MOVE", "QKC"])
    #expect(list.coins[0].thin && !list.coins[1].thin)
    #expect(list.coins[1].trendVsBTC == nil && list.coins[1].contract == nil)
    #expect(account.openPositions.count == 1 && account.openPositions[0].stake == 200)
    #expect(account.venueName == "Testnet")
}

@MainActor
@Test func filtersEventsByOrdersAndWatchlist() async {
    let now = Date()
    let api = FakeAPI(
        events: [
            event("order", category: .trade, kind: "вхід", at: now, symbols: ["ABC"]),
            event("blur", category: .short, kind: "Monitoring Tag Binance", at: now, symbols: ["BLUR"]),
            event("hack", category: .incident, kind: "Злам", at: now),
        ],
        coins: [
            WatchCoin(
                asset: "BLUR", note: nil, added: nil, known: true, reasons: ["monitoring-tag"], tagged: true,
                price: nil,
                volume: 6e5, volumeDay: nil, change1d: nil, change7d: nil, trendVsBTC: nil, contract: nil,
                funding: nil, openInterest: nil)
        ]
    )
    let store = AppStore(api: api)

    await store.refresh()

    #expect(store.filtered(.all).count == 3)
    #expect(store.filtered(.orders).map(\.id) == ["order"])
    #expect(store.filtered(.watchlist).map(\.id) == ["blur"])
}

@MainActor
@Test func actionsReachTheServerAndKeepItsRefusal() async {
    let refused = FakeAPI(refusal: "позиції немає")
    let store = AppStore(api: refused)

    await store.close(Position.sample)

    #expect(refused.calls.entered == ["close MOVEUSDT-1"] && store.actionError == "позиції немає")

    store.connect(to: FakeAPI())

    await store.close(Position.sample)

    #expect(store.actionError == nil)
}

@MainActor
@Test func searchFindsCoinsByTickerPrefixFirst() async {
    let names = ["ARB", "BAR", "BARD", "SOL"]
    let coins = names.map {
        WatchCoin(
            asset: $0, note: nil, added: nil, known: true, reasons: [], tagged: false, price: nil, volume: 1,
            volumeDay: nil, change1d: nil, change7d: nil, trendVsBTC: nil, contract: nil, funding: nil,
            openInterest: nil
        )
    }
    let store = AppStore(api: FakeAPI(coins: coins))

    await store.refresh()

    #expect(store.search("bar").map(\.asset) == ["BAR", "BARD", "ARB"].filter { $0.contains("BAR") })
    #expect(store.search(" ").count == 4)
}

@MainActor @Test func theWatchlistShowsTheCoinsPulseOrItsPriceAlone() async {
    func coin(_ asset: String, price: Double?) -> WatchCoin {
        WatchCoin(
            asset: asset, note: nil, added: nil, known: true, reasons: [], tagged: false, price: price, volume: 1e6,
            volumeDay: nil, change1d: nil, change7d: nil, trendVsBTC: nil, contract: nil, funding: nil,
            openInterest: nil
        )
    }

    var api = FakeAPI()
    api.board = PulseBoard(coins: [PulseCoin(asset: "XMR", contract: "XMRUSDT", price: 558, change1h: -0.03)])
    let store = AppStore(api: api)

    await store.refresh()

    let board = store.board(of: [coin("XMR", price: 557), coin("BARD", price: 0.085)])

    #expect(board.coins.map(\.asset) == ["XMR", "BARD"])
    #expect(board.coins[0].change1h == -0.03 && board.coins[0].contract == "XMRUSDT")
    #expect(board.coins[1].price == 0.085 && board.coins[1].contract.isEmpty)
}

@Test func eventsWithOneIDAreShownOnce() {
    let now = Date()
    let events = [
        event("a", category: .trade, kind: "план · змінено", at: now),
        event("a", category: .trade, kind: "план · змінено", at: now.addingTimeInterval(-10)),
        event("b", category: .trade, kind: "план · стежу", at: now.addingTimeInterval(-20)),
    ]

    #expect(AppStore.unique(events).map(\.id) == ["a", "b"])
}

@Test func theFeedIsShownNewestFirstAndKeepsTheServersOrderWithinOneMoment() {
    let now = Date()
    let events = [
        event("exit", category: .trade, kind: "вихід", at: now),
        event("fired", category: .trade, kind: "план · спрацював", at: now.addingTimeInterval(-82)),
        event("entry", category: .trade, kind: "вхід", at: now.addingTimeInterval(-28)),
        event("one", category: .short, kind: "делістинг", at: now.addingTimeInterval(-100)),
        event("two", category: .short, kind: "делістинг", at: now.addingTimeInterval(-100)),
    ]

    #expect(AppStore.newestFirst(events).map(\.id) == ["exit", "entry", "fired", "one", "two"])
}

@MainActor
@Test func theHeavyReadsRepeatLessOftenThanTheLightOnes() async {
    let api = FakeAPI()
    let store = AppStore(api: api, pulseEvery: 3600, slowEvery: 3600)

    await store.refresh()
    await store.refresh()
    await store.refresh()

    #expect(api.calls.coinReads == 1 && api.calls.pulseReads == 1)

    store.connect(to: api)
    await store.refresh()

    #expect(api.calls.coinReads == 2 && api.calls.pulseReads == 2)
}

@MainActor @Test func theFeedKeepsOnlyThePlansStillWatchedAsTriggers() async {
    let plan = Scenario(condition: "ціна нижче 0.0424", action: "шорт", risk: .medium, why: "пробій")
    var api = FakeAPI()
    api.watched.items = [
        PlanWatch(id: "a", asset: "MEGA", plan: plan, state: .armed),
        PlanWatch(id: "b", asset: "XMR", plan: plan, state: .refuted),
    ]
    let store = AppStore(api: api)

    await store.refresh()

    #expect(store.triggers.map(\.id) == ["a"])
    #expect(store.filtered(.triggers).isEmpty)
}

@MainActor @Test func theResultsOfThePlansAndTradesAreReadFromTheServer() async {
    let store = AppStore(api: FakeAPI())

    await store.refresh()

    #expect(store.results?.planLine == "Планів 7: стежу 2 · увійшли 1 · відхилено 3 · минув строк 0 · знято 1")
    #expect(store.results?.tradeLine == "Угод закрито 1: у плюс 1, у мінус 0, разом +$0.63 · відкрито 0")
    #expect(DeskResults().planLine == "Планів на стеженні ще не було.")
    #expect(DeskResults().tradeLine == "Угод ще не було.")
}

@MainActor @Test func autoTradingIsSwitchedOnTheServerAndReadBack() async {
    let api = FakeAPI()
    let store = AppStore(api: api)

    #expect(store.auto == nil)

    await store.refresh()

    #expect(store.auto == false)

    await store.setAuto(true)

    #expect(api.calls.auto && store.auto == true)
}

@MainActor @Test func aCoinIsFoundByItsTickerWithItsTag() async {
    var api = FakeAPI()
    api.coins = [
        WatchCoin(
            asset: "WIF", note: nil, added: nil, known: true, reasons: [], tagged: true, price: 0.15, volume: 1e6,
            volumeDay: nil, change1d: nil, change7d: nil, trendVsBTC: nil, contract: "WIFUSDT", funding: nil,
            openInterest: nil
        )
    ]
    let store = AppStore(api: api)

    #expect(store.coin("WIF") == nil)

    await store.refresh()

    #expect(store.coin("WIF")?.tagged == true && store.coin("BONK") == nil)
}

@Test func anEventOfAKindTheAppDoesNotKnowIsShownAsANote() throws {
    let stored = """
        [{"id": "listing:1", "source": "exchange-listings", "category": "listing", "kind": "Coinbase · лістинг",
          "title": "Coinbase: лістинг GRASS", "time": "2026-11-13T17:30:00Z"},
         {"id": "trade:1", "source": "trader", "category": "trade", "kind": "вхід", "title": "Лонг BTC",
          "time": "2026-11-13T16:36:48Z"}]
        """
    let events = try ServerJSON.decoder.decode([BotEvent].self, from: Data(stored.utf8))

    #expect(events.map(\.category) == [.incident, .trade])
}
