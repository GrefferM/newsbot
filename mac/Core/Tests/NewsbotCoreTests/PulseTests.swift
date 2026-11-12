import Foundation
import Testing

@testable import NewsbotCore

@MainActor
@Test func thePulseBoardPutsMoversFirstAndSortsByAnyColumn() async throws {
    let json = """
        {"at": "2026-11-11T08:01:36Z", "btc_1h": 0.002, "coins": [
          {"asset": "BTC", "contract": "BTCUSDT", "price": 73000, "change_15m": 0.001, "change_1h": 0.002, "change_4h": 0.01,
           "change_24h": -0.02, "volume_1h": 9e8, "volume_surge": 0.9, "buy_share": 0.5, "range_position": 0.5,
           "funding_8h": 0.0001, "headlines": []},
          {"asset": "ERA", "contract": "ERAUSDT", "price": 0.5, "change_15m": -0.01, "change_1h": -0.065, "change_4h": -0.02,
           "change_24h": 0.04, "volume_1h": 3e6, "volume_surge": 2.8, "buy_share": 0.48, "range_position": 0.2,
           "funding_8h": 0.0001, "woke": {"minutes": 20, "interest_5m": -0.03, "longs_15m": 0.025}, "headlines": []},
          {"asset": "ARIA", "contract": "ARIAUSDT", "price": 0.23, "change_15m": 0.02, "change_1h": 0.075, "change_4h": 0.08,
           "change_24h": 0.23, "volume_1h": 5e6, "volume_surge": 3.5, "buy_share": 0.55, "range_position": 0.98,
           "funding_8h": 0.0019, "woke": {"minutes": 5, "interest_5m": 0.04, "longs_15m": -0.02},
           "headlines": [{"published": "2026-11-11T07:30:00Z", "title": "Aria lists on a new venue", "outlet": "X", "url": "u"}]}
        ]}
        """
    let board = try ServerJSON.decoder.decode(PulseBoard.self, from: Data(json.utf8))
    let store = AppStore(api: FakeAPI(board: board))

    await store.refresh()

    #expect(store.pulse.movers == 2 && store.pulse.coins.count == 3)
    #expect(PulseSort.activity.sorted(store.pulse.coins, ascending: false).map(\.asset) == ["ARIA", "ERA", "BTC"])
    #expect(PulseSort.hour.sorted(store.pulse.coins, ascending: true).map(\.asset) == ["ERA", "BTC", "ARIA"])
    #expect(PulseSort.surge.sorted(store.pulse.coins, ascending: false).first?.asset == "ARIA")
    #expect(store.pulse.coins[2].reason == "X: Aria lists on a new venue" && store.pulse.coins[1].reason == nil)
    #expect(store.pulse.coins[1].caption != nil)
    #expect(store.pulse.coins[2].caption?.hasSuffix("Aria lists on a new venue") == true)
    #expect(store.pulse.coins[0].caption == nil)
    #expect(store.pulse(of: "ERA")?.active == true && store.pulse(of: "DOGE") == nil)
}

@Test func thePulseIsSearchedByTicker() {
    let board = PulseBoard(coins: [PulseCoin(asset: "INJ"), PulseCoin(asset: "AIO"), PulseCoin(asset: "PEPE")])

    #expect(board.found(" in").map(\.asset) == ["INJ"])
    #expect(board.found("").count == 3)
}

@Test func anIdeaCarriesItsEventAndThePulseVerdict() throws {
    let json = """
        {"asset": "ZEC", "direction": "шорт", "why": "падає", "cancel": "вище 1100", "price": 1050,
         "funding_8h": 0.0001, "event": "злам мосту — Видання", "woke_up": true}
        """
    let idea = try ServerJSON.decoder.decode(Idea.self, from: Data(json.utf8))

    #expect(idea.event == "злам мосту — Видання" && idea.facts?.hasSuffix("ожила в пульсі") == true)
}

@Test func aPulseCoinCarriesItsDayVolume() throws {
    let json = """
        {"asset": "ARIA", "contract": "ARIAUSDT", "price": 0.25, "change_15m": 0, "change_1h": 0.08, "change_4h": 0.1,
         "change_24h": 0.3, "volume_1h": 48000000, "volume_surge": 5.9, "buy_share": 0.53, "funding_8h": 0.00105,
         "volume_24h": 310000000, "woke": {"minutes": 0, "interest_5m": 0.03, "longs_15m": 0.02}, "headlines": []}
        """
    let coin = try ServerJSON.decoder.decode(PulseCoin.self, from: Data(json.utf8))

    #expect(coin.volumeDay == 310_000_000 && coin.funding == 0.00105)
}

@Test func aMarkedCoinTellsItsStageAndASteadyTrendItsLength() {
    let early = PulseCoin(asset: "KGEN", woke: PulseWake(minutes: 0, interest: 0.031, longs: -0.024))
    let late = PulseCoin(asset: "BOME", woke: PulseWake(minutes: 35, interest: -0.052, longs: 0.03))
    let steady = PulseCoin(asset: "GRAM", trendHours: 72, trendMove: 0.2)

    let fewer = Format.percent(-0.024, digits: 1).dropLast()
    let closed = Format.percent(-0.052, digits: 1)

    #expect(early.active && early.stage == "ожила щойно: інтерес +3.1%, лонги \(fewer) п.п.")
    #expect(late.active && late.stage == "ожила 35 хв тому: інтерес \(closed), лонги +3.0 п.п.")
    #expect(!steady.active && steady.stage == nil && steady.trend == "стала тенденція 3 д: +20%")
    #expect(PulseBoard(coins: [early, late, steady]).movers == 2)
    let risen = PulseCoin(asset: "AKE", launch: PulseLaunch(hours: 2, move: 0.246, times: 34.6))
    let sorted = PulseSort.activity.sorted([steady, PulseCoin(asset: "AAA"), late, early, risen], ascending: false)

    #expect(sorted.map(\.asset) == ["AKE", "KGEN", "BOME", "GRAM", "AAA"])
    #expect(risen.active && risen.stage == "зліт із тиші 2 год тому: +25% за годину, ×35 звичайної")
    #expect(PulseLaunch(hours: 0, move: 0.12, times: 20).told.hasPrefix("зліт із тиші зараз: +12%"))
}
