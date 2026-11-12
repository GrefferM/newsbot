import Foundation
import Observation

@MainActor
@Observable
public final class CoinNews {
    public let researcher: Conversation
    public private(set) var news: [Finding] = []
    public private(set) var loaded = false
    public private(set) var failure: String?

    public init(asset: String) {
        researcher = Conversation(agent: .osinter, topic: asset)
    }

    public var searching: Bool {
        researcher.working
    }

    public func load(from api: BotAPI) async {
        await reload(from: api)
        await researcher.restore(from: api)
    }

    public func search(_ query: String, with api: BotAPI) async {
        let focus = query.trimmingCharacters(in: .whitespaces)
        let text = focus.isEmpty ? Self.everything : focus

        guard await researcher.send(text, with: api) != nil else {
            return
        }

        await reload(from: api)
    }

    public func sorted(newestFirst: Bool) -> [Finding] {
        news.sorted { newestFirst ? $0.stamp > $1.stamp : $0.stamp < $1.stamp }
    }

    public func forget(_ finding: Finding, with api: BotAPI) async {
        do {
            let result = try await api.forget(news: finding.id)

            if result.ok {
                news.removeAll { $0.id == finding.id }
            }

            failure = result.error
        } catch {
            failure = String(describing: error)
        }
    }

    private func reload(from api: BotAPI) async {
        if let kept = try? await api.news(asset: researcher.topic) {
            news = kept
            loaded = true
        }
    }

    public static let everything = "Знайди новини про неї за останній місяць і події, заплановані наперед."
}
