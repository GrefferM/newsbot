import Foundation

public struct DepthPoint: Identifiable, Equatable, Sendable {
    public let offset: Double
    public let total: Double
    public let bid: Bool

    public var id: String {
        "\(bid) \(offset)"
    }
}

public struct Wall: Equatable, Sendable {
    public let offset: Double
    public let value: Double
}

public struct Liquidity: Equatable, Sendable {
    public let spread: Double
    public let bidNear: Double
    public let askNear: Double
    public let bidFar: Double
    public let askFar: Double
    public let slippage: Double?
    public let buySlippage: Double?
    public let bidWall: Wall?
    public let askWall: Wall?
    public let curve: [DepthPoint]

    public static let near = 0.01
    public static let far = 0.02

    public init?(book: OrderBook, stake: Double) {
        guard let bid = book.bids.first?.price, let ask = book.asks.first?.price, bid > 0, ask >= bid else {
            return nil
        }

        let mid = (bid + ask) / 2
        let bids = book.bids.filter { $0.price >= mid * (1 - Self.far) }
        let asks = book.asks.filter { $0.price <= mid * (1 + Self.far) }

        spread = (ask - bid) / mid
        bidNear = Self.depth(book.bids, from: mid * (1 - Self.near), to: mid)
        askNear = Self.depth(book.asks, from: mid, to: mid * (1 + Self.near))
        bidFar = Self.depth(bids, from: 0, to: mid)
        askFar = Self.depth(asks, from: mid, to: .infinity)
        slippage = Self.fill(stake, from: book.bids).map { (mid - $0) / mid }
        buySlippage = Self.fill(stake, from: book.asks).map { ($0 - mid) / mid }
        bidWall = Self.wall(bids, mid: mid)
        askWall = Self.wall(asks, mid: mid)
        curve = Self.stairs(bids, mid: mid, bid: true).reversed() + Self.stairs(asks, mid: mid, bid: false)
    }

    public func bidShare(near: Bool) -> Double? {
        let bid = near ? bidNear : bidFar
        let both = bid + (near ? askNear : askFar)

        return both > 0 ? bid / both : nil
    }

    static func depth(_ levels: [BookLevel], from low: Double, to high: Double) -> Double {
        levels.filter { $0.price >= low && $0.price <= high }.reduce(0) { $0 + $1.value }
    }

    static func fill(_ stake: Double, from levels: [BookLevel]) -> Double? {
        var left = stake
        var quantity = 0.0

        for level in levels {
            let taken = min(left, level.value)

            quantity += taken / level.price
            left -= taken

            if left <= 0 {
                return stake / quantity
            }
        }

        return nil
    }

    static func wall(_ levels: [BookLevel], mid: Double) -> Wall? {
        levels.max { $0.value < $1.value }.map { Wall(offset: $0.price / mid - 1, value: $0.value) }
    }

    static func stairs(_ levels: [BookLevel], mid: Double, bid: Bool) -> [DepthPoint] {
        var total = 0.0
        var out = [DepthPoint(offset: 0, total: 0, bid: bid)]

        for level in levels {
            total += level.value

            out.append(DepthPoint(offset: level.price / mid - 1, total: total, bid: bid))
        }

        return out
    }
}
