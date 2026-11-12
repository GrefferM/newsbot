import Foundation

public protocol BotAPI: Sendable {
    func watchlist() async throws -> Watchlist
    func coins() async throws -> Watchlist
    func pulse() async throws -> PulseBoard
    func tradfi() async throws -> PulseBoard
    func calendar() async throws -> [MarketEvent]
    func keep(asset: String, note: String) async throws -> ActionResult
    func drop(asset: String) async throws -> ActionResult
    func events(limit: Int) async throws -> [BotEvent]
    func forget(event id: String) async throws -> ActionResult
    func book() async throws -> Book
    func health() async throws -> Health
    func close(position id: String) async throws -> ActionResult
    func ask(_ agent: Agent, question: String, about topic: String) async throws -> AgentAnswer
    func scout(question: String) async throws -> AgentAnswer
    func ideas(question: String) async throws -> AgentAnswer
    func history(of agent: Agent, about topic: String, limit: Int) async throws -> [Exchange]
    func forget(_ agent: Agent, about topic: String) async throws -> ActionResult
    func news(asset: String) async throws -> [Finding]
    func forget(news id: String) async throws -> ActionResult
    func plan(asset: String) async throws -> EntryPlan?
    func keep(_ plan: EntryPlan, asset: String) async throws -> ActionResult
    func watches(asset: String) async throws -> [PlanWatch]
    func triggers() async throws -> [PlanWatch]
    func results() async throws -> DeskResults
    func auto() async throws -> AutoTrading
    func setAuto(_ wanted: AutoTrading) async throws -> ActionResult
    func watch(_ plan: Scenario, asset: String) async throws -> ActionResult
    func unwatch(_ id: String) async throws -> ActionResult
    func amend(_ id: String, plan: Scenario) async throws -> ActionResult
}
