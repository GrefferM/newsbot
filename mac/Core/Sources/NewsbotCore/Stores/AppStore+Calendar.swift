import Foundation

extension AppStore {
    public func calendar(tradfi wanted: Bool, now: Date = Date()) -> [MarketEvent] {
        let contracts = Set(tradfi.coins.map(\.asset))
        let ahead = calendar.filter { $0.approximate != true && $0.ahead(of: now) }

        return Array(ahead.filter { $0.traded(among: contracts) == wanted }.prefix(6))
    }

    public func traded(_ asset: String) -> Bool {
        tradfi.coins.contains { $0.asset == asset }
    }

    public func dates(of asset: String, now: Date = Date()) -> [MarketEvent] {
        let traded = traded(asset)

        return Array(calendar.filter { $0.concerns(asset, traded: traded) && $0.ahead(of: now) }.prefix(8))
    }

    public func marks(of asset: String) -> [MarketEvent] {
        let traded = traded(asset)

        return calendar.filter { $0.approximate != true && $0.concerns(asset, traded: traded) }
    }
}
