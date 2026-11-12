import Foundation

public struct SeriesPoint: Identifiable, Equatable, Sendable {
    public let time: Date
    public let value: Double

    public var id: Date {
        time
    }

    public init(time: Date, value: Double) {
        self.time = time
        self.value = value
    }
}

public struct Derivatives: Equatable, Sendable {
    public let openInterest: [SeriesPoint]
    public let funding: [SeriesPoint]
    public let longs: [SeriesPoint]

    public static let points = 500
    public static let history: TimeInterval = 30 * 86_400

    public init(openInterest: [SeriesPoint], funding: [SeriesPoint], longs: [SeriesPoint]) {
        self.openInterest = openInterest
        self.funding = funding
        self.longs = longs
    }

    public var openInterestChange: Double? {
        guard let first = openInterest.first?.value, let last = openInterest.last?.value, first > 0 else {
            return nil
        }

        return last / first - 1
    }

    public func within(_ span: ClosedRange<Date>) -> Derivatives {
        Derivatives(
            openInterest: openInterest.filter { span.contains($0.time) },
            funding: funding.filter { span.contains($0.time) },
            longs: longs.filter { span.contains($0.time) }
        )
    }

    public var empty: Bool {
        openInterest.isEmpty && funding.isEmpty && longs.isEmpty
    }
}

extension Array where Element == SeriesPoint {
    public func nearest(to time: Date) -> SeriesPoint? {
        self.min { abs($0.time.timeIntervalSince(time)) < abs($1.time.timeIntervalSince(time)) }
    }
}
