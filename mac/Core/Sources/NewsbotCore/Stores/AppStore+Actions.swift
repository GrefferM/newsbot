import Foundation

extension AppStore {
    public func keep(_ asset: String, note: String) async {
        await act { try await self.api.keep(asset: asset, note: note) }
    }

    public func switchStrategy(_ id: String, on: Bool) async {
        await act { try await self.api.switchStrategy(id, to: StrategySwitch(on: on)) }
    }

    public func tune(_ id: String, rule: Data) async -> String? {
        await act { try await self.api.tune(id, rule: rule) }

        return actionError
    }

    public func unwatch(_ trigger: PlanWatch) async {
        await act { try await self.api.unwatch(trigger.id) }
    }

    public func drop(_ asset: String) async {
        await act { try await self.api.drop(asset: asset) }
    }

    public func forget(_ event: BotEvent) async {
        await act { try await self.api.forget(event: event.id) }
    }

    public func close(_ position: Position) async {
        await act { try await self.api.close(position: position.id) }
    }

    func act(_ action: () async throws -> ActionResult) async {
        do {
            let result = try await action()
            actionError = result.error
        } catch {
            actionError = String(describing: error)
        }

        await refresh()
    }
}
