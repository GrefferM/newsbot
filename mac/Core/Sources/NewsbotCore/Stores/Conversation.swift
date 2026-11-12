import Foundation
import Observation

@MainActor
@Observable
public final class Conversation {
    public let agent: Agent
    public let topic: String
    public private(set) var turns: [ChatTurn] = []
    public private(set) var working = false
    public private(set) var failure: String?

    public static let kept = 10

    public init(agent: Agent, topic: String = "") {
        self.agent = agent
        self.topic = topic
    }

    public func restore(from api: BotAPI) async {
        guard turns.isEmpty, !working, let past = try? await api.history(of: agent, about: topic, limit: Self.kept)
        else {
            return
        }

        turns = past.map { exchange in
            ChatTurn(
                at: exchange.at, question: exchange.question, answer: exchange.answer, failed: exchange.failed == true,
                calls: exchange.calls, ideas: exchange.ideas ?? [])
        }
    }

    @discardableResult
    public func send(_ text: String, with api: BotAPI, reach: Reach = .ask) async -> AgentAnswer? {
        let turn = ChatTurn(at: Date(), question: text)

        turns = Array((turns + [turn]).suffix(Self.kept))
        working = true
        failure = nil

        defer {
            working = false
        }

        do {
            let answer = try await reply(to: text, with: api, reach: reach)

            settle(turn.id) { told in
                told.answer = answer.text
                told.calls = answer.calls
                told.ideas = answer.ideas ?? []
            }

            return answer
        } catch {
            settle(turn.id) { told in
                told.answer = String(describing: error)
                told.failed = true
            }

            return nil
        }
    }

    private func reply(to text: String, with api: BotAPI, reach: Reach) async throws -> AgentAnswer {
        switch reach {
        case .ask:
            try await api.ask(agent, question: text, about: topic)
        case .scout:
            try await api.scout(question: text)
        case .ideas:
            try await api.ideas(question: text)
        }
    }

    public func clear(with api: BotAPI) async {
        do {
            let result = try await api.forget(agent, about: topic)

            if result.ok {
                turns = []
            }

            failure = result.error
        } catch {
            failure = String(describing: error)
        }
    }

    private func settle(_ id: UUID, _ change: (inout ChatTurn) -> Void) {
        guard let index = turns.firstIndex(where: { $0.id == id }) else {
            return
        }

        change(&turns[index])
    }
}
