import Foundation

public struct Exchange: Decodable, Equatable, Sendable {
    public let at: Date
    public let question: String
    public let answer: String
    public let calls: [String]
    public let ideas: [Idea]?
    public let failed: Bool?

    public init(
        at: Date, question: String, answer: String, calls: [String] = [], ideas: [Idea]? = nil, failed: Bool? = nil
    ) {
        self.at = at
        self.question = question
        self.answer = answer
        self.calls = calls
        self.ideas = ideas
        self.failed = failed
    }
}
