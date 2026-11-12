import Foundation
import Testing

@testable import NewsbotCore

final class Calls: @unchecked Sendable {
    var entered: [String] = []
    var coinReads = 0
    var pulseReads = 0
    var auto = false
}

final class Watched: @unchecked Sendable {
    var items: [PlanWatch] = []
}

struct FakeAPI: BotAPI {
    var events: [BotEvent] = []
    var coins: [WatchCoin] = []
    var calls = Calls()
    var refusal: String?
    var answers: [Agent: AgentAnswer] = [:]
    var pinned: [Finding] = []

    func news(asset: String) async throws -> [Finding] {
        pinned
    }

    var past: [Exchange] = []

    func scout(question: String) async throws -> AgentAnswer {
        AgentAnswer(text: "Воронка: " + question)
    }

    func ideas(question: String) async throws -> AgentAnswer {
        AgentAnswer(
            text: "Ринок падає разом.",
            ideas: [
                Idea(
                    asset: "ZEC", direction: "шорт", why: "−8% за годину на обігу ×4", cancel: "вище 1100", price: 1050,
                    funding: -0.0001)
            ])
    }

    func history(of agent: Agent, about topic: String, limit: Int) async throws -> [Exchange] {
        past
    }

    func forget(_ agent: Agent, about topic: String) async throws -> ActionResult {
        ActionResult(ok: true, error: nil)
    }

    func forget(news id: String) async throws -> ActionResult {
        ActionResult(ok: true, error: nil)
    }

    var fixed: EntryPlan?

    func plan(asset: String) async throws -> EntryPlan? {
        fixed
    }

    var watched = Watched()

    func watches(asset: String) async throws -> [PlanWatch] {
        watched.items
    }

    func triggers() async throws -> [PlanWatch] {
        watched.items.filter { $0.state == .armed }
    }

    func auto() async throws -> AutoTrading {
        AutoTrading(enabled: calls.auto)
    }

    func setAuto(_ wanted: AutoTrading) async throws -> ActionResult {
        calls.auto = wanted.enabled

        return ActionResult(ok: true, error: nil)
    }

    func results() async throws -> DeskResults {
        DeskResults(
            plans: .init(armed: 2, fired: 1, refuted: 3, cancelled: 1),
            trades: .init(open: 0, closed: 1, won: 1, lost: 0, pnl: 0.63))
    }

    func watch(_ plan: Scenario, asset: String) async throws -> ActionResult {
        watched.items.append(PlanWatch(id: "w\(watched.items.count + 1)", asset: asset, plan: plan, state: .armed))

        return ActionResult(ok: true, error: nil)
    }

    func amend(_ id: String, plan: Scenario) async throws -> ActionResult {
        guard let index = watched.items.firstIndex(where: { $0.id == id }) else {
            return ActionResult(ok: false, error: "такого стеження немає")
        }

        let old = watched.items[index]
        watched.items[index] = PlanWatch(id: id, asset: old.asset, plan: plan, state: .armed)

        return ActionResult(ok: true, error: nil)
    }

    func unwatch(_ id: String) async throws -> ActionResult {
        watched.items.removeAll { $0.id == id }

        return ActionResult(ok: true, error: nil)
    }

    func keep(_ plan: EntryPlan, asset: String) async throws -> ActionResult {
        ActionResult(ok: true, error: nil)
    }

    func ask(_ agent: Agent, question: String, about topic: String) async throws -> AgentAnswer {
        guard let answer = answers[agent] else {
            throw AgentError.failed("немає")
        }

        return answer
    }

    func watchlist() async throws -> Watchlist {
        Watchlist(built: nil, coins: coins)
    }

    func coins() async throws -> Watchlist {
        calls.coinReads += 1

        return Watchlist(built: nil, coins: coins)
    }

    var board = PulseBoard.empty

    func pulse() async throws -> PulseBoard {
        calls.pulseReads += 1

        return board
    }

    func tradfi() async throws -> PulseBoard {
        .empty
    }

    var dates: [MarketEvent] = []

    func calendar() async throws -> [MarketEvent] {
        dates
    }

    func keep(asset: String, note: String) async throws -> ActionResult {
        calls.entered.append("keep " + asset)

        return ActionResult(ok: true, error: nil)
    }

    func drop(asset: String) async throws -> ActionResult {
        calls.entered.append("drop " + asset)

        return ActionResult(ok: true, error: nil)
    }

    func events(limit: Int) async throws -> [BotEvent] {
        events
    }

    func forget(event id: String) async throws -> ActionResult {
        ActionResult(ok: true, error: nil)
    }

    func book() async throws -> Book {
        .empty
    }

    func health() async throws -> Health {
        Health(started: Date(), sources: [:])
    }

    func close(position id: String) async throws -> ActionResult {
        calls.entered.append("close " + id)

        return ActionResult(ok: refusal == nil, error: refusal)
    }
}

func event(_ id: String, category: EventCategory, kind: String, at time: Date, symbols: [String] = []) -> BotEvent {
    BotEvent(
        id: id,
        source: "test",
        category: category,
        kind: kind,
        title: id,
        url: nil,
        time: time,
        symbols: symbols,
        details: nil
    )
}

extension Position {
    static let sample = Position(
        id: "MOVEUSDT-1", symbol: "MOVE", contract: "MOVEUSDT", side: .short, kind: "Повний делістинг Binance",
        size: 0.2, stake: 200, quantity: 200, entry: 1, stop: 1.3, target: 0.7,
        opened: Date(), deadline: Date(), open: true, exit: nil, closedAt: nil, reason: nil, result: nil, pnl: nil
    )
}
