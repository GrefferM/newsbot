import Foundation

public struct AutoTrading: Codable, Equatable, Sendable {
    public let strategies: [AutoStrategy]

    public init(strategies: [AutoStrategy]) {
        self.strategies = strategies
    }

    public var running: Bool {
        strategies.contains { $0.on }
    }

    public var countLine: String {
        "увімкнено \(strategies.filter(\.on).count) з \(strategies.count)"
    }
}

public struct AutoStrategy: Codable, Equatable, Sendable, Identifiable {
    public let id: String
    public let name: String
    public let about: String
    public let terms: [StrategyTerm]
    public let on: Bool
    public let record: StrategyRecord
    public let rule: [String: Double]

    public init(
        id: String, name: String, about: String, terms: [StrategyTerm] = [], on: Bool,
        record: StrategyRecord = StrategyRecord(), rule: [String: Double] = [:]
    ) {
        self.id = id
        self.name = name
        self.about = about
        self.terms = terms
        self.on = on
        self.record = record
        self.rule = rule
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)

        id = try container.decode(String.self, forKey: .id)
        name = try container.decode(String.self, forKey: .name)
        about = try container.decode(String.self, forKey: .about)
        terms = try container.decodeIfPresent([StrategyTerm].self, forKey: .terms) ?? []
        on = try container.decode(Bool.self, forKey: .on)
        record = try container.decodeIfPresent(StrategyRecord.self, forKey: .record) ?? StrategyRecord()
        rule = try container.decodeIfPresent([String: Double].self, forKey: .rule) ?? [:]
    }
}

public struct StrategyTerm: Codable, Equatable, Sendable, Identifiable {
    public let name: String
    public let value: String

    public init(name: String, value: String) {
        self.name = name
        self.value = value
    }

    public var id: String {
        name
    }
}

public struct StrategyRecord: Codable, Equatable, Sendable {
    public let open: Int
    public let closed: Int
    public let won: Int
    public let pnl: Double

    public init(open: Int = 0, closed: Int = 0, won: Int = 0, pnl: Double = 0) {
        self.open = open
        self.closed = closed
        self.won = won
        self.pnl = pnl
    }

    enum CodingKeys: String, CodingKey {
        case open, closed, won
        case pnl = "pnl_usd"
    }

    public var line: String {
        guard closed > 0 else {
            return open > 0 ? "Відкрито угод: \(open), закритих ще немає" : "Угод ще не було"
        }

        let share = Int((Double(won) / Double(closed) * 100).rounded())
        let opened = open > 0 ? " · відкрито \(open)" : ""

        return "Закрито \(closed): у плюс \(won) (\(share)%), разом \(Format.money(pnl))\(opened)"
    }
}

public struct StrategySwitch: Codable, Equatable, Sendable {
    public let on: Bool

    public init(on: Bool) {
        self.on = on
    }
}
