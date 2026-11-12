import Foundation

public struct ChatTurn: Identifiable, Equatable, Sendable {
    public let id = UUID()
    public let at: Date
    public let question: String
    public var answer: String?
    public var failed: Bool
    public var calls: [String]
    public var ideas: [Idea]

    public init(
        at: Date, question: String, answer: String? = nil, failed: Bool = false, calls: [String] = [],
        ideas: [Idea] = []
    ) {
        self.at = at
        self.question = question
        self.answer = answer
        self.failed = failed
        self.calls = calls
        self.ideas = ideas
    }
}
