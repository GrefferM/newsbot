import Foundation

public struct DayStats: Equatable, Sendable {
    public let change: Double
    public let volume: Double

    public init(change: Double, volume: Double) {
        self.change = change
        self.volume = volume
    }
}

public protocol MarketFeed: Sendable {
    func candles(contract: String, interval: ChartInterval, limit: Int, before: Date?) async throws -> [Candle]
    func updates(contract: String, interval: ChartInterval) -> AsyncThrowingStream<Candle, Error>
    func day(contract: String) async throws -> DayStats
    func price(contract: String, testnet: Bool) async throws -> Double
    func book(contract: String) async throws -> OrderBook
    func derivatives(contract: String, interval: ChartInterval, span: ClosedRange<Date>?) async throws -> Derivatives
}
