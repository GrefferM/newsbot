import Foundation

public struct Idea: Decodable, Identifiable, Equatable, Sendable {
    public let asset: String
    public let direction: String
    public let why: String
    public let cancel: String
    public let price: Double?
    public let funding: Double?
    public let event: String?
    public let woke: Bool?

    public init(
        asset: String, direction: String, why: String, cancel: String, price: Double? = nil, funding: Double? = nil,
        event: String? = nil, woke: Bool? = nil
    ) {
        self.asset = asset
        self.direction = direction
        self.why = why
        self.cancel = cancel
        self.price = price
        self.funding = funding
        self.event = event
        self.woke = woke
    }

    public var facts: String? {
        guard let price, price > 0 else {
            return nil
        }

        let mark = woke == true ? " · ожила в пульсі" : ""

        return "ціна \(Format.price(price)) · фандинг \(Format.funding(funding ?? 0)) / 8 год" + mark
    }

    enum CodingKeys: String, CodingKey {
        case asset, direction, why, cancel, price, event
        case funding = "funding_8h"
        case woke = "woke_up"
    }

    public var id: String {
        asset
    }

    public var long: Bool {
        direction == "лонг"
    }
}

public enum Reach: Sendable {
    case ask
    case scout
    case ideas
}
