import Foundation

public enum WatchState: String, Decodable, Sendable {
    case armed, fired, refuted, expired, cancelled

    public var name: String {
        switch self {
        case .armed:
            "стежу"
        case .fired:
            "спрацював"
        case .refuted:
            "спростовано"
        case .expired:
            "термін минув"
        case .cancelled:
            "знято"
        }
    }
}

public struct PlanWatch: Decodable, Identifiable, Equatable, Sendable {
    public let id: String
    public let asset: String
    public let plan: Scenario
    public let state: WatchState
    public let note: String?
    public let armed: Date
    public let expires: Date
    public let checks: Int
    public let notBefore: Date?

    public init(
        id: String, asset: String, plan: Scenario, state: WatchState, note: String? = nil, armed: Date = Date(),
        expires: Date = Date(), checks: Int = 0, notBefore: Date? = nil
    ) {
        self.id = id
        self.asset = asset
        self.plan = plan
        self.state = state
        self.note = note
        self.armed = armed
        self.expires = expires
        self.checks = checks
        self.notBefore = notBefore
    }

    public func waiting(now: Date) -> String {
        var parts = ["Стеження до \(Format.day(expires))"]

        if let notBefore, notBefore > now {
            parts.append("перевірка не раніше \(Format.day(notBefore)) \(Format.clock(notBefore))")
        }

        if checks > 0 {
            parts.append("агента питали: \(checks)")
        }

        return parts.joined(separator: " · ")
    }

    enum CodingKeys: String, CodingKey {
        case id, asset, plan, state, note, armed, expires, checks
        case notBefore = "not_before"
    }
}
