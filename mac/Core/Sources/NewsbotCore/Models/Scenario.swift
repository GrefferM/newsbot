import Foundation

public enum RiskLevel: String, Codable, Sendable {
    case minimal = "мінімальний"
    case medium = "середній"
    case high = "високий"

    public var weight: Int {
        switch self {
        case .minimal:
            0
        case .medium:
            1
        case .high:
            2
        }
    }
}

public struct Scenario: Codable, Identifiable, Equatable, Sendable {
    public let condition: String
    public let action: String
    public let risk: RiskLevel
    public let why: String
    public let direction: String?
    public let wakeAbove: Double?
    public let wakeBelow: Double?
    public let sizePct: Double?
    public let stop: Double?
    public let take: Double?
    public let wakeAfter: String?

    public init(
        condition: String, action: String, risk: RiskLevel, why: String, direction: String? = nil,
        wakeAbove: Double? = nil, wakeBelow: Double? = nil, sizePct: Double? = nil, stop: Double? = nil,
        take: Double? = nil, wakeAfter: String? = nil
    ) {
        self.condition = condition
        self.action = action
        self.risk = risk
        self.why = why
        self.direction = direction
        self.wakeAbove = wakeAbove
        self.wakeBelow = wakeBelow
        self.sizePct = sizePct
        self.stop = stop
        self.take = take
        self.wakeAfter = wakeAfter
    }

    public var id: String {
        condition
    }

    public var armable: Bool {
        (direction == "шорт" || direction == "лонг") && triggered && (sizePct ?? 0) > 0
            && (stop ?? 0) > 0 && (take ?? 0) > 0
    }

    private var triggered: Bool {
        (wakeAbove ?? wakeBelow ?? 0) > 0
    }

    public var numbers: String? {
        guard armable, let sizePct, let stop, let take else {
            return nil
        }

        let level = wakeAbove.map { "від \(Format.price($0))" } ?? wakeBelow.map { "до \(Format.price($0))" }
        let date = wakeAfter.map { "з \(Format.stamp($0))" }
        let wake = [level, date].compactMap { $0 }.joined(separator: " ")

        let size = String(format: sizePct == sizePct.rounded() ? "%.0f%%" : "%.1f%%", sizePct)

        return
            "\(direction ?? "") \(wake) · \(size) рахунку · стоп \(Format.price(stop)) · тейк \(Format.price(take))"
    }

    enum CodingKeys: String, CodingKey {
        case condition, action, risk, why, direction
        case wakeAbove = "wake_price_above"
        case wakeBelow = "wake_price_below"
        case sizePct = "size_pct"
        case stop = "stop_price"
        case take = "take_price"
        case wakeAfter = "wake_after"
    }
}
