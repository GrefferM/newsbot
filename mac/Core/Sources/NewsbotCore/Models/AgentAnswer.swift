import Foundation

public struct AgentAnswer: Decodable, Equatable, Sendable {
    public let text: String
    public let findings: [Finding]
    public let scenarios: [Scenario]
    public let calls: [String]
    public let ideas: [Idea]?

    public init(
        text: String, findings: [Finding] = [], scenarios: [Scenario] = [], calls: [String] = [],
        ideas: [Idea]? = nil
    ) {
        self.text = text
        self.findings = findings
        self.scenarios = scenarios
        self.calls = calls
        self.ideas = ideas
    }
}
