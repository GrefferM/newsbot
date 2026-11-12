import Foundation

extension AppStore {
    public func search(_ text: String) -> [WatchCoin] {
        let query = text.trimmingCharacters(in: .whitespaces).uppercased()

        guard !query.isEmpty else {
            return coins.coins
        }

        return coins.coins
            .filter { $0.asset.contains(query) }
            .sorted { ($0.asset.hasPrefix(query) ? 0 : 1, $0.asset) < ($1.asset.hasPrefix(query) ? 0 : 1, $1.asset) }
    }

    public func keeps(_ asset: String) -> Bool {
        watchlist.coins.contains { $0.asset == asset }
    }

    public func board(of coins: [WatchCoin]) -> PulseBoard {
        PulseBoard(
            at: pulse.at,
            coins: coins.map { coin in
                pulse(of: coin.asset) ?? PulseCoin(asset: coin.asset, price: coin.price ?? 0)
            }
        )
    }

    public func pulse(of asset: String) -> PulseCoin? {
        pulsed[asset]
    }

    public func coin(_ asset: String) -> WatchCoin? {
        scanned[asset]
    }

    func index() {
        keep(\.scanned, Dictionary((watchlist.coins + coins.coins).map { ($0.asset, $0) }) { first, _ in first })
    }
}
