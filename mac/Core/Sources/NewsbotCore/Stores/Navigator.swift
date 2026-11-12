import Foundation
import Observation

@MainActor
@Observable
public final class Navigator {
    public private(set) var coin: String
    public private(set) var news: String?

    private let defaults: UserDefaults

    public init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
        self.coin = defaults.string(forKey: Self.coinKey) ?? ""
    }

    public func show(coin asset: String, news text: String? = nil) {
        coin = asset
        news = text

        defaults.set(asset, forKey: Self.coinKey)
    }

    public func shown(news text: String) {
        if news == text {
            news = nil
        }
    }

    private static let coinKey = "shownCoin"
}
