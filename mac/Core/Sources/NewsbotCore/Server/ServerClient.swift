import Foundation

public enum ClientError: Error, Equatable {
    case status(Int)
    case held
}

public struct ServerClient: BotAPI {
    private let base: URL
    private let session: URLSession

    public init(base: URL, session: URLSession = .shared) {
        self.base = base
        self.session = session
    }

    public func watchlist() async throws -> Watchlist {
        try await get("watchlist")
    }

    public func coins() async throws -> Watchlist {
        try await get("coins")
    }

    public func pulse() async throws -> PulseBoard {
        try await get("pulse")
    }

    public func tradfi() async throws -> PulseBoard {
        try await get("tradfi")
    }

    public func calendar() async throws -> [MarketEvent] {
        try await get("calendar")
    }

    public func keep(asset: String, note: String) async throws -> ActionResult {
        let body = try JSONSerialization.data(withJSONObject: ["asset": asset, "note": note])

        return try await send("watchlist", method: "POST", body: body)
    }

    public func drop(asset: String) async throws -> ActionResult {
        try await send("watchlist/\(asset)", method: "DELETE")
    }

    public func events(limit: Int) async throws -> [BotEvent] {
        try await get("events", query: [URLQueryItem(name: "limit", value: String(limit))])
    }

    public func book() async throws -> Book {
        try await get("positions")
    }

    public func health() async throws -> Health {
        try await get("healthz")
    }

    public func ask(_ agent: Agent, question: String, about topic: String) async throws -> AgentAnswer {
        try await answer(to: "agents/\(agent.rawValue)", ["question": question, "topic": topic], within: 600)
    }

    public func scout(question: String) async throws -> AgentAnswer {
        try await answer(to: "scout", ["question": question], within: 1800)
    }

    public func ideas(question: String) async throws -> AgentAnswer {
        try await answer(to: "ideas", ["question": question], within: 600)
    }

    private func answer(to path: String, _ body: [String: String], within seconds: TimeInterval) async throws
        -> AgentAnswer
    {
        var request = URLRequest(url: base.appending(path: path))
        request.httpMethod = "POST"
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        request.timeoutInterval = seconds

        let (data, response) = try await session.data(for: request)

        if let http = response as? HTTPURLResponse, http.statusCode != 200 {
            let refusal = try? ServerJSON.decoder.decode(ActionResult.self, from: data)

            throw AgentError.failed(refusal?.error ?? "HTTP \(http.statusCode)")
        }

        return try ServerJSON.decoder.decode(AgentAnswer.self, from: data)
    }

    public func history(of agent: Agent, about topic: String, limit: Int) async throws -> [Exchange] {
        try await get(
            "agents/\(agent.rawValue)/history",
            query: [URLQueryItem(name: "topic", value: topic), URLQueryItem(name: "limit", value: String(limit))]
        )
    }

    public func forget(_ agent: Agent, about topic: String) async throws -> ActionResult {
        var url = base.appending(path: "agents/\(agent.rawValue)/history")
        url.append(queryItems: [URLQueryItem(name: "topic", value: topic)])

        var request = URLRequest(url: url)
        request.httpMethod = "DELETE"

        let (data, _) = try await session.data(for: request)

        return try ServerJSON.decoder.decode(ActionResult.self, from: data)
    }

    public func news(asset: String) async throws -> [Finding] {
        try await get("coins/\(asset)/news")
    }

    public func forget(event id: String) async throws -> ActionResult {
        try await send("events", method: "DELETE", query: [URLQueryItem(name: "id", value: id)])
    }

    public func forget(news id: String) async throws -> ActionResult {
        try await send("news/\(id)", method: "DELETE")
    }

    public func plan(asset: String) async throws -> EntryPlan? {
        let (data, response) = try await session.data(from: planURL(asset))

        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            return nil
        }

        return try ServerJSON.decoder.decode(EntryPlan.self, from: data)
    }

    public func keep(_ plan: EntryPlan, asset: String) async throws -> ActionResult {
        let encoder = JSONEncoder()
        encoder.dateEncodingStrategy = .iso8601

        var request = URLRequest(url: planURL(asset))
        request.httpMethod = "PUT"
        request.httpBody = try encoder.encode(plan)

        let (data, _) = try await session.data(for: request)

        return try ServerJSON.decoder.decode(ActionResult.self, from: data)
    }

    public func watches(asset: String) async throws -> [PlanWatch] {
        try await get("coins/\(asset)/watches")
    }

    public func triggers() async throws -> [PlanWatch] {
        try await get("watches")
    }

    public func watch(_ plan: Scenario, asset: String) async throws -> ActionResult {
        try await send(plan, to: "coins/\(asset)/watches", method: "POST")
    }

    public func amend(_ id: String, plan: Scenario) async throws -> ActionResult {
        try await send(plan, to: "watches/\(id)", method: "PUT")
    }

    public func results() async throws -> DeskResults {
        try await get("results")
    }

    public func auto() async throws -> AutoTrading {
        try await get("auto")
    }

    public func setAuto(_ wanted: AutoTrading) async throws -> ActionResult {
        try await send(wanted, to: "auto", method: "PUT")
    }

    private func send<Body: Encodable>(_ body: Body, to path: String, method: String) async throws -> ActionResult {
        var request = URLRequest(url: base.appending(path: path))
        request.httpMethod = method
        request.httpBody = try JSONEncoder().encode(body)

        let (data, response) = try await session.data(for: request)

        if let http = response as? HTTPURLResponse, http.statusCode == 200 {
            return ActionResult(ok: true, error: nil)
        }

        return try ServerJSON.decoder.decode(ActionResult.self, from: data)
    }

    public func unwatch(_ id: String) async throws -> ActionResult {
        try await send("watches/\(id)", method: "DELETE")
    }

    private func planURL(_ asset: String) -> URL {
        base.appending(path: "coins/\(asset)/plan")
    }

    public func close(position id: String) async throws -> ActionResult {
        try await post("positions/\(id)/close")
    }

    private func get<T: Decodable>(_ path: String, query: [URLQueryItem] = []) async throws -> T {
        var url = base.appending(path: path)

        if !query.isEmpty {
            url.append(queryItems: query)
        }

        let (data, response) = try await session.data(from: url)

        if let http = response as? HTTPURLResponse, http.statusCode != 200 {
            throw ClientError.status(http.statusCode)
        }

        return try ServerJSON.decoder.decode(T.self, from: data)
    }

    private func post(_ path: String) async throws -> ActionResult {
        try await send(path, method: "POST")
    }

    private func send(
        _ path: String, method: String, body: Data? = nil, query: [URLQueryItem] = []
    ) async throws -> ActionResult {
        var request = URLRequest(url: base.appending(path: path).appending(queryItems: query))
        request.httpMethod = method
        request.httpBody = body

        let (data, _) = try await session.data(for: request)

        return try ServerJSON.decoder.decode(ActionResult.self, from: data)
    }
}
