import Foundation
import Observation

@MainActor
@Observable
public final class PositionPrice {
    public private(set) var price: Double?

    private let feed: MarketFeed
    private let every: Duration

    public init(feed: MarketFeed, every: Duration = .seconds(5)) {
        self.feed = feed
        self.every = every
    }

    public func follow(
        _ position: @escaping @MainActor () -> Position?, testnet: @escaping @MainActor () -> Bool
    ) async {
        while !Task.isCancelled {
            await read(position(), testnet: testnet())

            try? await Task.sleep(for: every)
        }
    }

    public func read(_ position: Position?, testnet: Bool) async {
        guard let position, position.open else {
            return
        }

        price = try? await feed.price(contract: position.contract, testnet: testnet)
    }
}
