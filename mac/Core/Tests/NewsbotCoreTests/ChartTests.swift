import Foundation
import Testing

@testable import NewsbotCore

func bar(_ minute: Double, close: Double) -> Candle {
    Candle(time: Date(timeIntervalSince1970: minute * 60), open: 1, high: 2, low: 0.5, close: close, volume: 10)
}

struct FakeFeed: MarketFeed {
    var history: [Candle]
    var live: [Candle]

    func candles(contract: String, interval: ChartInterval, limit: Int, before: Date?) async throws -> [Candle] {
        guard let before else {
            return Array(history.suffix(limit))
        }

        return Array(history.filter { $0.time < before }.suffix(limit))
    }

    func updates(contract: String, interval: ChartInterval) -> AsyncThrowingStream<Candle, Error> {
        AsyncThrowingStream { continuation in
            for candle in live {
                continuation.yield(candle)
            }

            continuation.finish()
        }
    }

    func day(contract: String) async throws -> DayStats {
        DayStats(change: -0.04, volume: 1_500_000)
    }

    func price(contract: String, testnet: Bool) async throws -> Double {
        1
    }

    func book(contract: String) async throws -> OrderBook {
        OrderBook(bids: [], asks: [])
    }

    func derivatives(contract: String, interval: ChartInterval, span: ClosedRange<Date>?) async throws -> Derivatives {
        Derivatives(openInterest: [], funding: [], longs: [])
    }
}

@Test func readsFuturesCandlesAndStreamUpdates() throws {
    let rows = """
        [[1762905600000, "0.0130", "0.0141", "0.0122", "0.0125", "1", 0, "612345.6", 1, "0", "0", "0"]]
        """
    let update = """
        {"e":"kline","s":"MOVEUSDT","k":{"t":1762905660000,"o":"0.0125","h":"0.0127","l":"0.0124","c":"0.0126","q":"9000.5","x":false}}
        """

    let candles = try BinanceFutures.parseCandles(Data(rows.utf8))
    let live = BinanceFutures.parseUpdate(update)

    #expect(candles.count == 1 && candles[0].high == 0.0141 && !candles[0].rising && candles[0].volume == 612_345.6)
    #expect(live?.close == 0.0126 && live?.volume == 9000.5)
    #expect(live?.time.timeIntervalSince(candles[0].time) == 60)
    #expect(BinanceFutures.parseUpdate("{\"result\":null}") == nil)
}

@Test func mergingReplacesTheOpenCandleAndAppendsANewOne() {
    let history = [bar(1, close: 1), bar(2, close: 1.1)]

    let same = ChartModel.merge(history, bar(2, close: 1.2), limit: 3)
    let next = ChartModel.merge(same, bar(3, close: 1.3), limit: 3)
    let capped = ChartModel.merge(next, bar(4, close: 1.4), limit: 3)
    let stale = ChartModel.merge(capped, bar(1, close: 9), limit: 3)

    #expect(same.map(\.close) == [1, 1.2])
    #expect(next.map(\.close) == [1, 1.2, 1.3])
    #expect(capped.map(\.close) == [1.2, 1.3, 1.4])
    #expect(stale == capped)
    #expect(ChartModel.merge([], bar(1, close: 1), limit: 3).count == 1)
}

@MainActor
@Test func theChartLoadsHistoryThenFollowsTheStream() async {
    let feed = FakeFeed(
        history: [bar(1, close: 1), bar(2, close: 1.1)], live: [bar(2, close: 1.15), bar(3, close: 1.2)])
    let model = ChartModel(feed: feed)

    await model.watch(contract: "MOVEUSDT", interval: .minute)

    #expect(model.candles.map(\.close) == [1, 1.15, 1.2])
    #expect(model.price == 1.2 && model.day?.change == -0.04 && model.failure == nil)
}

@Test func liquidityMeasuresSpreadDepthAndSlippage() {
    let book = OrderBook(
        bids: [
            BookLevel(price: 0.99, quantity: 100), BookLevel(price: 0.985, quantity: 200),
            BookLevel(price: 0.97, quantity: 1000),
        ],
        asks: [BookLevel(price: 1.01, quantity: 50), BookLevel(price: 1.03, quantity: 500)]
    )

    let liquidity = Liquidity(book: book, stake: 150)

    #expect(abs((liquidity?.spread ?? 0) - 0.02) < 1e-9)
    #expect(liquidity?.bidNear == 99 && liquidity?.askNear == 50.5)
    #expect(abs((liquidity?.bidFar ?? 0) - 296) < 1e-9 && liquidity?.askFar == 50.5)
    #expect(abs((liquidity?.slippage ?? 0) - (1 - 150 / (100 + 51 / 0.985))) < 1e-9)
    #expect(abs((liquidity?.buySlippage ?? 0) - (150 / (50 + 99.5 / 1.03) - 1)) < 1e-9)
    #expect(abs((liquidity?.bidShare(near: true) ?? 0) - 99 / 149.5) < 1e-9)
    #expect(abs((liquidity?.bidWall?.value ?? 0) - 197) < 1e-9 && abs((liquidity?.bidWall?.offset ?? 0) + 0.015) < 1e-9)
    #expect(liquidity?.askWall?.value == 50.5 && liquidity?.curve.map(\.bid) == [true, true, true, false, false])
    #expect(liquidity?.curve.first?.total == 296 && liquidity?.curve.last?.total == 50.5)
    #expect(Liquidity(book: book, stake: 10_000)?.slippage == nil)
    #expect(Liquidity(book: OrderBook(bids: [], asks: []), stake: 150) == nil)
}

@Test func readsTheBookAndTheDerivativeSeries() throws {
    let depth = """
        {"bids":[["0.0089","1000"],["0.0088","2500"]],"asks":[["0.0090","700"]]}
        """
    let interest = """
        [{"symbol":"X","sumOpenInterestValue":"1000000","timestamp":1762905600000},
         {"symbol":"X","sumOpenInterestValue":"1410000","timestamp":1762992000000}]
        """

    let book = try BinanceFutures.parseBook(Data(depth.utf8))
    let series = BinanceFutures.parseSeries(Data(interest.utf8), time: "timestamp", value: "sumOpenInterestValue")
    let derivatives = Derivatives(openInterest: series, funding: [], longs: [])

    #expect(book.bids.count == 2 && book.asks.first?.price == 0.009)
    #expect(series.count == 2 && abs((derivatives.openInterestChange ?? 0) - 0.41) < 1e-9)
    #expect(derivatives.within(series[1].time...series[1].time.addingTimeInterval(60)).openInterest == [series[1]])
    #expect(derivatives.within(series[1].time.addingTimeInterval(60)...series[1].time.addingTimeInterval(90)).empty)
    #expect(series.nearest(to: series[0].time.addingTimeInterval(50_000)) == series[1])
}

@Test func aHostThatSaidTooManyRequestsIsLeftAloneForAtLeastAMinute() {
    let brake = Brake()
    let now = Date()

    #expect(!brake.holds("fapi.binance.com", now: now))

    brake.hold("fapi.binance.com", asked: 5, now: now)

    #expect(brake.holds("fapi.binance.com", now: now.addingTimeInterval(59)))
    #expect(!brake.holds("fapi.binance.com", now: now.addingTimeInterval(61)))
    #expect(!brake.holds("testnet.binancefuture.com", now: now))

    brake.hold("fapi.binance.com", asked: 300, now: now)

    #expect(brake.holds("fapi.binance.com", now: now.addingTimeInterval(299)))
}

@Test func theChartFrameLeavesRoomForAComingEventAndKeepsOnlyTheMarksInView() throws {
    let start = try #require(ServerJSON.parseTime("2026-11-13T10:00:00Z"))
    let candles = (0..<4).map { step in
        Candle(
            time: start.addingTimeInterval(Double(step) * 300), open: 1, high: 1.2, low: 0.9, close: 1.1,
            volume: 10)
    }
    let last = start.addingTimeInterval(900)
    let marks = [
        ChartMark(time: start.addingTimeInterval(-86_400), labels: ["давно"], ahead: false),
        ChartMark(time: start.addingTimeInterval(300), labels: ["тег"], ahead: false),
        ChartMark(time: last.addingTimeInterval(1_800), labels: ["за 8 хв", "CPI"], ahead: true),
    ]

    let frame = ChartFrame(candles: candles, interval: .fiveMinutes, marks: marks, levels: [2], now: last)

    #expect(frame.domain.lowerBound == start.addingTimeInterval(-300))
    #expect(frame.domain.upperBound == last.addingTimeInterval(2_400))
    #expect(frame.marks.map(\.label) == ["тег", "за 8 хв  ·  CPI"])
    #expect(frame.prices.upperBound > 2 && frame.prices.lowerBound < 0.9)
    #expect(frame.candle(at: start.addingTimeInterval(320), in: candles) == candles[1])
    #expect(frame.candle(at: start.addingTimeInterval(-86_400), in: candles) == nil)

    let quiet = ChartFrame(candles: candles, interval: .fiveMinutes, marks: [], levels: [], now: last)

    #expect(quiet.domain.upperBound == last.addingTimeInterval(900))
}

@Test func theChartsAxisNamesHoursAndMarksTheTurnOfADay() throws {
    let utc = try #require(TimeZone(identifier: "UTC"))
    let start = try #require(ServerJSON.parseTime("2026-11-11T19:20:00Z"))
    let end = try #require(ServerJSON.parseTime("2026-11-12T05:20:00Z"))

    let ticks = ChartAxis.ticks(from: start, to: end, zone: utc)

    #expect(ticks.map(\.label) == ["20:00", "22:00", "12.11", "02:00", "04:00"])
    #expect(ticks.map(\.major) == [false, false, true, false, false])

    let days = ChartAxis.ticks(from: start, to: start.addingTimeInterval(120 * 86_400), zone: utc)

    #expect((5...9).contains(days.count) && days.allSatisfy { !$0.major && $0.label.count == 5 })
    #expect(days[1].time.timeIntervalSince(days[0].time) == 14 * 86_400)
}

@Test func aCandleUnderThePointerIsReadOut() throws {
    let utc = try #require(TimeZone(identifier: "UTC"))
    let at = try #require(ServerJSON.parseTime("2026-11-12T14:00:00Z"))
    let first = Candle(time: at, open: 0.0312, high: 0.0320, low: 0.0305, close: 0.0309, volume: 1_240_000)
    let second = Candle(
        time: at.addingTimeInterval(3600), open: 0.0309, high: 0.0311, low: 0.03, close: 0.0302, volume: 9)

    let range = try #require([first, second].prices(with: [0.04], margin: 0.1))

    #expect(abs(range.lowerBound - 0.029) < 1e-9 && abs(range.upperBound - 0.041) < 1e-9)
    #expect([Candle]().prices() == nil)
    #expect([first, second].nearest(to: at.addingTimeInterval(1700)) == first)
    #expect([first, second].nearest(to: at.addingTimeInterval(1900)) == second)
    let said = "12.11 14:00  ·  відкр. 0.031200  ·  макс. 0.032000  ·  мін. 0.030500  ·  закр. 0.030900"

    #expect(first.readout(zone: utc) == said + "  ·  −0.96%  ·  обіг $1.24M")
}

@Test func theChartMarksPastEventsOnTheirCandleAndAComingOneOnlyTenMinutesAhead() throws {
    let utc = try #require(TimeZone(identifier: "UTC"))
    let now = try #require(ServerJSON.parseTime("2026-11-13T11:07:00Z"))
    let at = { (minutes: Double) in now.addingTimeInterval(minutes * 60) }
    let dated = [
        MarketEvent(id: "old", time: at(-63), title: "Розблокування", source: "AGT", kind: "unlock", assets: ["AGT"]),
        MarketEvent(id: "soon", time: at(7.5), title: "CPI", source: "США"),
        MarketEvent(id: "later", time: at(40), title: "FOMC", source: "ФРС"),
        MarketEvent(id: "day", time: at(-300), title: "Голосування", source: "AGT", kind: "coin", allDay: true),
    ]
    let seen = [
        BotEvent(
            id: "tag", source: "binance", category: .short, kind: "тег", title: "Monitoring Tag", url: nil,
            time: at(-64), symbols: ["AGT"], details: nil)
    ]

    let marks = ChartMarks.marks(dated: dated, seen: seen, interval: .fiveMinutes, now: now, zone: utc)

    #expect(marks.count == 2 && marks[0].ahead == false && marks[1].ahead)
    #expect(marks[0].time == ServerJSON.parseTime("2026-11-13T10:00:00Z"))
    #expect(marks[0].labels == ["13.11 10:04 AGT: Розблокування", "13.11 10:03 тег: Monitoring Tag"])
    #expect(marks[1].time == at(7.5) && marks[1].label == "за 8 хв  ·  США: CPI")

    let daily = ChartMarks.marks(dated: dated, seen: [], interval: .day, now: now, zone: utc)

    #expect(daily.first?.time == ServerJSON.parseTime("2026-11-13T00:00:00Z") && daily.first?.labels.count == 2)
}

struct BookFeed: MarketFeed {
    var depth = OrderBook(bids: [BookLevel(price: 0.99, quantity: 500)], asks: [BookLevel(price: 1.01, quantity: 500)])
    var failing = false

    func candles(contract: String, interval: ChartInterval, limit: Int, before: Date?) async throws -> [Candle] {
        []
    }

    func updates(contract: String, interval: ChartInterval) -> AsyncThrowingStream<Candle, Error> {
        AsyncThrowingStream { $0.finish() }
    }

    func day(contract: String) async throws -> DayStats {
        DayStats(change: 0, volume: 0)
    }

    func price(contract: String, testnet: Bool) async throws -> Double {
        testnet ? 2 : 68722
    }

    func book(contract: String) async throws -> OrderBook {
        if failing {
            throw URLError(.badServerResponse)
        }

        return depth
    }

    func derivatives(contract: String, interval: ChartInterval, span: ClosedRange<Date>?) async throws -> Derivatives {
        if failing {
            throw URLError(.badServerResponse)
        }

        return Derivatives(openInterest: [], funding: [], longs: [])
    }
}

@MainActor @Test func aCoinsMarketReadsItsBookAndItsDerivativesAndSaysWhenBinanceFails() async {
    let market = CoinMarket(feed: BookFeed())

    await market.readBook(contract: "ZECUSDT", stake: 200)
    await market.loadDerivatives(contract: "ZECUSDT", interval: .hour, span: nil)

    #expect(market.liquidity != nil && !market.bookFailed)
    #expect(market.derivatives != nil && !market.derivativesFailed)

    let down = CoinMarket(feed: BookFeed(failing: true))

    await down.readBook(contract: "ZECUSDT", stake: 200)
    await down.loadDerivatives(contract: "ZECUSDT", interval: .hour, span: nil)

    #expect(down.liquidity == nil && down.bookFailed && down.derivativesFailed)
}

@MainActor @Test func anOpenPositionIsPricedOnItsVenueAndAClosedOneIsNot() async {
    let live = PositionPrice(feed: BookFeed())

    await live.read(nil, testnet: false)
    #expect(live.price == nil)

    await live.read(Position.sample, testnet: false)
    #expect(live.price == 68722)

    await live.read(Position.sample, testnet: true)
    #expect(live.price == 2)
}

@MainActor
@Test func theChartIsPulledBackIntoHistoryAndReturnsToTheLiveEdge() async {
    let history = (1...700).map { bar(Double($0), close: Double($0)) }
    let model = ChartModel(feed: FakeFeed(history: history, live: [bar(701, close: 701)]))

    await model.watch(contract: "MOVEUSDT", interval: .minute)

    #expect(model.candles.count == ChartModel.limit + 1 && model.shown.last?.close == 701 && model.latest?.close == 701)

    await model.pan(to: 50)

    #expect(model.back == 50 && model.shown.count == ChartModel.limit && model.shown.last?.close == 651)
    #expect(model.candles.first?.close == 81 && model.latest == nil, "older candles are fetched before they are needed")

    await model.pan(to: 10_000)

    #expect(model.back == model.candles.count - ChartModel.limit && model.shown.first == model.candles.first)

    await model.pan(to: 10_000)
    await model.pan(to: 10_000)

    #expect(model.candles.first?.close == 1 && model.shown.first?.close == 1, "the pull ends where the history does")

    await model.pan(to: -5)

    #expect(model.back == 0 && model.shown.last?.close == 701 && model.price == 701)
}

@Test func aPulledChartKeepsItsPlaceAndDerivativesFollowItsWindow() {
    let candles = (1...10).map { bar(Double($0), close: Double($0)) }

    #expect(ChartModel.window(candles, back: 3, limit: 4).map(\.close) == [4, 5, 6, 7])
    #expect(ChartModel.window(candles, back: 0, limit: 4).map(\.close) == [7, 8, 9, 10])
    #expect(ChartModel.window(candles, back: 50, limit: 4).isEmpty)
    #expect(
        ChartModel.joined(candles, [bar(10, close: 10.5), bar(11, close: 11)]).map(\.close).suffix(3) == [9, 10.5, 11])

    let shown = Date(timeIntervalSince1970: 100_000)...Date(timeIntervalSince1970: 107_200)

    #expect(CoinMarket.reach(of: shown, back: 0, interval: .minute) == nil)
    #expect(
        CoinMarket.reach(of: shown, back: 60, interval: .minute)
            == Date(timeIntervalSince1970: 96_400)...Date(timeIntervalSince1970: 110_800))
    #expect(ChartInterval.minute.derivativesStep.name == "5m" && ChartInterval.day.derivativesStep.seconds == 86_400)
}
