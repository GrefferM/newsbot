import Foundation
import Observation

@MainActor
@Observable
public final class CoinMarket {
    public private(set) var liquidity: Liquidity?
    public private(set) var bookFailed = false
    public private(set) var derivatives: Derivatives?
    public private(set) var derivativesFailed = false

    private let feed: MarketFeed
    private let every: Duration
    private let fresh: Duration

    public init(feed: MarketFeed, every: Duration = .seconds(6), fresh: Duration = .seconds(300)) {
        self.feed = feed
        self.every = every
        self.fresh = fresh
    }

    public func followBook(contract: String, stake: @escaping @MainActor () -> Double) async {
        liquidity = nil

        while !Task.isCancelled {
            await readBook(contract: contract, stake: stake())

            try? await Task.sleep(for: every)
        }
    }

    public func readBook(contract: String, stake: Double) async {
        do {
            liquidity = Liquidity(book: try await feed.book(contract: contract), stake: stake)
            bookFailed = false
        } catch {
            bookFailed = true
        }
    }

    public func followDerivatives(contract: String, interval: ChartInterval, span: ClosedRange<Date>?) async {
        await loadDerivatives(contract: contract, interval: interval, span: span)

        while span == nil && !Task.isCancelled {
            try? await Task.sleep(for: fresh)

            await loadDerivatives(contract: contract, interval: interval, span: nil)
        }
    }

    public func loadDerivatives(contract: String, interval: ChartInterval, span: ClosedRange<Date>?) async {
        do {
            let found = try await feed.derivatives(contract: contract, interval: interval, span: span)

            if Task.isCancelled {
                return
            }

            derivatives = found
            derivativesFailed = false
        } catch {
            derivativesFailed = !Task.isCancelled && derivatives == nil
        }
    }

    public func forgetDerivatives() {
        derivatives = nil
    }

    public nonisolated static func reach(
        of shown: ClosedRange<Date>?, back: Int, interval: ChartInterval
    ) -> ClosedRange<Date>? {
        guard back > 0, let shown else {
            return nil
        }

        let margin = interval.seconds * Double(ChartModel.limit) / 2

        return shown.lowerBound.addingTimeInterval(-margin)...shown.upperBound.addingTimeInterval(margin)
    }
}
