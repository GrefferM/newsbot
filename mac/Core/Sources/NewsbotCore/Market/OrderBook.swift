import Foundation

public struct BookLevel: Equatable, Sendable {
    public let price: Double
    public let quantity: Double

    public init(price: Double, quantity: Double) {
        self.price = price
        self.quantity = quantity
    }

    var value: Double {
        price * quantity
    }
}

public struct OrderBook: Equatable, Sendable {
    public let bids: [BookLevel]
    public let asks: [BookLevel]

    public init(bids: [BookLevel], asks: [BookLevel]) {
        self.bids = bids
        self.asks = asks
    }
}
