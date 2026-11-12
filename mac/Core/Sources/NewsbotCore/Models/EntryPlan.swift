import Foundation
import Observation

public struct EntryPlan: Codable, Equatable, Sendable {
    public let wish: String?
    public let summary: String
    public let scenarios: [Scenario]
    public let made: Date

    public init(wish: String? = nil, summary: String, scenarios: [Scenario], made: Date) {
        self.wish = wish
        self.summary = summary
        self.scenarios = scenarios
        self.made = made
    }

    public var ranked: [Scenario] {
        scenarios.enumerated()
            .sorted { a, b in
                a.element.risk.weight == b.element.risk.weight
                    ? a.offset < b.offset : a.element.risk.weight > b.element.risk.weight
            }
            .map(\.element)
    }
}
