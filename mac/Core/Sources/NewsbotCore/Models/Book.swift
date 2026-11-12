import Foundation

public struct Position: Decodable, Identifiable, Equatable, Sendable {
    public let id: String
    public let symbol: String
    public let contract: String
    public let side: Side
    public let kind: String
    public let size: Double
    public let stake: Double
    public let quantity: Double?
    public let entry: Double
    public let stop: Double
    public let target: Double
    public let opened: Date
    public let deadline: Date
    public let open: Bool
    public let exit: Double?
    public let closedAt: Date?
    public let reason: String?
    public let result: Double?
    public let pnl: Double?

    enum CodingKeys: String, CodingKey {
        case id, symbol, contract, side, kind, size, stake, quantity, entry, stop, target, opened, deadline, open, exit,
            reason, pnl
        case closedAt = "closed_at"
        case result = "return"
    }
}

public struct Book: Decodable, Equatable, Sendable {
    public let venue: String
    public let start: Double
    public let equity: Double
    public let exposure: Double
    public let positions: [Position]

    public var openPositions: [Position] {
        positions.filter(\.open)
    }

    public var change: Double {
        start > 0 ? equity / start - 1 : 0
    }

    public var summary: String {
        change == 0 ? Format.money(equity) : "\(Format.money(equity)) · \(Format.percent(change, digits: 3))"
    }

    public var probe: Double {
        equity > 0 ? equity * 0.2 : 200
    }

    public var testnet: Bool {
        venue == "testnet"
    }

    public var venueName: String {
        switch venue {
        case "testnet":
            "Testnet"
        case "":
            "не підключено"
        default:
            venue
        }
    }

    public static let empty = Book(venue: "", start: 0, equity: 0, exposure: 0, positions: [])
}
