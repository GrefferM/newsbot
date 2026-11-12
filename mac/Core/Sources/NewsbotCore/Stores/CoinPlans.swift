import Foundation
import Observation

@MainActor
@Observable
public final class CoinPlans {
    public private(set) var plan: EntryPlan?
    public private(set) var watches: [PlanWatch] = []
    public private(set) var planning = false
    public private(set) var asking: String?
    public private(set) var failure: String?

    public let asset: String

    public init(asset: String) {
        self.asset = asset
    }

    public func load(from api: BotAPI) async {
        if let kept = try? await api.plan(asset: asset) {
            plan = kept
        }

        await reloadWatches(from: api)
    }

    public func make(_ wish: String, with api: BotAPI) async {
        let asked = wish.trimmingCharacters(in: .whitespaces)
        let question = "Склади плани входу." + (asked.isEmpty ? "" : " Побажання власника: \(asked)")

        planning = true
        asking = Self.shown(asked)
        failure = nil

        defer {
            planning = false
            asking = nil
        }

        do {
            let answer = try await api.ask(.risk, question: question, about: asset)
            let made = EntryPlan(
                wish: asked.isEmpty ? nil : asked, summary: answer.text, scenarios: answer.scenarios, made: Date())

            failure = try await api.keep(made, asset: asset).error
            plan = made
        } catch {
            failure = String(describing: error)
        }
    }

    nonisolated public static func shown(_ wish: String?) -> String {
        guard let wish, !wish.isEmpty else {
            return "Склади плани входу"
        }

        return wish
    }

    public func reloadWatches(from api: BotAPI) async {
        if let fresh = try? await api.watches(asset: asset) {
            watches = fresh
        }
    }

    public func watch(of scenario: Scenario) -> PlanWatch? {
        watches.last { $0.plan.condition == scenario.condition }
    }

    public var waiting: [PlanWatch] {
        watches.filter { $0.state == .armed }
    }

    public func settled(last count: Int) -> [PlanWatch] {
        Array(watches.filter { $0.state != .armed && $0.state != .cancelled }.suffix(count).reversed())
    }

    public func arm(_ scenario: Scenario, with api: BotAPI) async {
        await change(with: api) {
            try await api.watch(scenario, asset: asset)
        }
    }

    public func amend(_ watch: PlanWatch, to numbers: PlanNumbers, with api: BotAPI) async -> Bool {
        guard let plan = numbers.applied(to: watch.plan) else {
            failure = "Числа плану не читаються: перевір поріг, розмір, стоп і тейк."

            return false
        }

        await change(with: api) {
            try await api.amend(watch.id, plan: plan)
        }

        return failure == nil
    }

    public func disarm(_ watch: PlanWatch, with api: BotAPI) async {
        await change(with: api) {
            try await api.unwatch(watch.id)
        }
    }

    private func change(with api: BotAPI, _ action: () async throws -> ActionResult) async {
        do {
            failure = try await action().error
        } catch {
            failure = String(describing: error)
        }

        await reloadWatches(from: api)
    }
}
