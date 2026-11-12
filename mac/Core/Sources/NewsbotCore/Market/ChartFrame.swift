import Foundation

public struct ChartFrame: Equatable, Sendable {
    public let domain: ClosedRange<Date>
    public let prices: ClosedRange<Double>
    public let ticks: [TimeMark]
    public let marks: [ChartMark]

    public init(
        candles: [Candle], interval: ChartInterval, marks: [ChartMark], levels: [Double], now: Date,
        zone: TimeZone = .current
    ) {
        let step = interval.seconds
        let first = candles.first?.time ?? now.addingTimeInterval(-step * Double(ChartModel.limit))
        let last = candles.last?.time ?? now
        let room = last.addingTimeInterval(step * 3)
        let coming = marks.filter(\.ahead).map { $0.time.addingTimeInterval(step * 2) }.max() ?? room
        let domain = first.addingTimeInterval(-step)...max(room, coming)

        self.domain = domain
        self.prices = candles.prices(with: levels) ?? 0...1
        self.ticks = ChartAxis.ticks(from: domain.lowerBound, to: domain.upperBound, zone: zone)
        self.marks = marks.filter { domain.contains($0.time) }
    }

    public func candle(at moment: Date?, in candles: [Candle]) -> Candle? {
        moment.flatMap { domain.contains($0) ? candles.nearest(to: $0) : nil }
    }
}
