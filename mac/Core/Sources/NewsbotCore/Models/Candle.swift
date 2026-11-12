import Foundation

public struct Candle: Identifiable, Equatable, Sendable {
    public let time: Date
    public let open: Double
    public let high: Double
    public let low: Double
    public let close: Double
    public let volume: Double

    public var id: Date {
        time
    }

    public var rising: Bool {
        close >= open
    }

    public init(time: Date, open: Double, high: Double, low: Double, close: Double, volume: Double) {
        self.time = time
        self.open = open
        self.high = high
        self.low = low
        self.close = close
        self.volume = volume
    }
}

extension Candle {
    public var change: Double {
        open > 0 ? close / open - 1 : 0
    }

    public func opened(zone: TimeZone = .current) -> String {
        Format.dayTime(time, zone: zone)
    }

    public func readout(zone: TimeZone = .current) -> String {
        return [
            opened(zone: zone),
            "відкр. " + Format.price(open),
            "макс. " + Format.price(high),
            "мін. " + Format.price(low),
            "закр. " + Format.price(close),
            Format.percent(change, digits: 2),
            "обіг " + Format.usd(volume),
        ].joined(separator: "  ·  ")
    }
}

extension Array where Element == Candle {
    public func prices(with others: [Double] = [], margin: Double = 0.04) -> ClosedRange<Double>? {
        let all = map(\.low) + map(\.high) + others

        guard let low = all.min(), let high = all.max(), low > 0 else {
            return nil
        }

        let pad = Swift.max(high - low, high * 0.001) * margin

        return (low - pad)...(high + pad)
    }

    public func nearest(to time: Date) -> Candle? {
        self.min { abs($0.time.timeIntervalSince(time)) < abs($1.time.timeIntervalSince(time)) }
    }
}
