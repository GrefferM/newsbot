import Foundation

public struct PlanNumbers: Equatable, Sendable {
    public var wake: String
    public var after: String
    public var size: String
    public var stop: String
    public var take: String
    public let below: Bool

    public init(_ plan: Scenario) {
        below = plan.wakeBelow != nil && plan.wakeAbove == nil
        wake = Self.text(plan.wakeAbove ?? plan.wakeBelow)
        after = plan.wakeAfter ?? ""
        size = Self.text(plan.sizePct)
        stop = Self.text(plan.stop)
        take = Self.text(plan.take)
    }

    public var wakeTitle: String {
        below ? "Ціна-поріг, до" : "Ціна-поріг, від"
    }

    public func applied(to plan: Scenario) -> Scenario? {
        let date = after.trimmingCharacters(in: .whitespaces)

        guard let level = Self.number(wake), let size = Self.number(size), let stop = Self.number(stop),
            let take = Self.number(take)
        else {
            return nil
        }

        return Scenario(
            condition: plan.condition, action: plan.action, risk: plan.risk, why: plan.why, direction: plan.direction,
            wakeAbove: below ? nil : level, wakeBelow: below ? level : nil, sizePct: size, stop: stop, take: take,
            wakeAfter: date.isEmpty ? nil : date)
    }

    static func number(_ text: String) -> Double? {
        Double(text.trimmingCharacters(in: .whitespaces).replacing(",", with: "."))
    }

    static func text(_ value: Double?) -> String {
        guard let value else {
            return ""
        }

        let plain = String(value)

        return plain.hasSuffix(".0") ? String(plain.dropLast(2)) : plain
    }
}
