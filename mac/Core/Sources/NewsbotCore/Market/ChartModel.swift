import Foundation
import Observation

@MainActor
@Observable
public final class ChartModel {
    public nonisolated static let limit = 120
    public nonisolated static let page = 500
    public nonisolated static let kept = 3000

    public private(set) var candles: [Candle] = []
    public private(set) var back = 0
    public private(set) var day: DayStats?
    public private(set) var live = false
    public private(set) var failure: String?

    private let feed: MarketFeed
    private let retry: Duration
    private var watched: (contract: String, interval: ChartInterval)?
    private var aim = 0
    private var loading = false
    private var oldest = false

    public init(feed: MarketFeed, retry: Duration = .seconds(5)) {
        self.feed = feed
        self.retry = retry
    }

    public var price: Double? {
        candles.last?.close
    }

    public var shown: [Candle] {
        Self.window(candles, back: back, limit: Self.limit)
    }

    public var latest: Candle? {
        back == 0 ? candles.last : nil
    }

    public var span: ClosedRange<Date>? {
        guard let watched, let first = shown.first?.time, let last = shown.last?.time else {
            return nil
        }

        return first...last.addingTimeInterval(watched.interval.seconds * 3)
    }

    public func follow(contract: String, interval: ChartInterval) async {
        candles = []
        back = 0
        aim = 0
        oldest = false
        live = false
        watched = (contract, interval)

        while !Task.isCancelled {
            await watch(contract: contract, interval: interval)

            live = false

            try? await Task.sleep(for: retry)
        }
    }

    public func watch(contract: String, interval: ChartInterval) async {
        do {
            let fresh = try await feed.candles(contract: contract, interval: interval, limit: Self.limit, before: nil)

            candles = Self.joined(candles, fresh)
            watched = (contract, interval)
            day = try? await feed.day(contract: contract)
            failure = nil

            for try await update in feed.updates(contract: contract, interval: interval) {
                live = true
                take(update)
            }
        } catch is CancellationError {
            return
        } catch {
            failure = candles.isEmpty ? "Графік недоступний" : nil
        }
    }

    public func pan(to wanted: Int) async {
        aim = max(0, wanted)
        back = reachable(aim)

        if aim + Self.limit * 2 > candles.count {
            await older()

            back = reachable(aim)
        }
    }

    private func reachable(_ wanted: Int) -> Int {
        min(wanted, max(0, candles.count - Self.limit))
    }

    private func take(_ update: Candle) {
        let before = candles.count

        candles = Self.merge(candles, update, limit: Self.kept)

        if back > 0 && candles.count > before {
            back += 1
            aim += 1
        }
    }

    private func older() async {
        guard !loading, !oldest, candles.count < Self.kept, let watched, let first = candles.first else {
            return
        }

        loading = true

        defer {
            loading = false
        }

        let found = try? await feed.candles(
            contract: watched.contract, interval: watched.interval, limit: Self.page, before: first.time)

        guard let found, self.watched?.contract == watched.contract, self.watched?.interval == watched.interval else {
            return
        }

        let earlier = found.filter { $0.time < first.time }

        oldest = earlier.isEmpty
        candles = earlier + candles
    }

    public nonisolated static func window(_ candles: [Candle], back: Int, limit: Int) -> [Candle] {
        let end = max(0, candles.count - max(0, back))

        return Array(candles[max(0, end - limit)..<end])
    }

    public nonisolated static func joined(_ kept: [Candle], _ fresh: [Candle]) -> [Candle] {
        guard let first = fresh.first else {
            return kept
        }

        return kept.filter { $0.time < first.time } + fresh
    }

    public nonisolated static func merge(_ candles: [Candle], _ update: Candle, limit: Int) -> [Candle] {
        guard let last = candles.last else {
            return [update]
        }

        if update.time < last.time {
            return candles
        }

        if update.time == last.time {
            return candles.dropLast() + [update]
        }

        return Array((candles + [update]).suffix(limit))
    }
}
