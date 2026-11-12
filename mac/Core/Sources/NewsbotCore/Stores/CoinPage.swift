import Foundation
import Observation

@MainActor
@Observable
public final class CoinPage {
    public let asset: String
    public let news: CoinNews
    public let plans: CoinPlans
    public let trader: Conversation
    public let chart = ChartModel(feed: BinanceFutures())
    public let market = CoinMarket(feed: BinanceFutures())

    public init(asset: String) {
        self.asset = asset
        news = CoinNews(asset: asset)
        plans = CoinPlans(asset: asset)
        trader = Conversation(agent: .trader, topic: asset)
    }

    public func load(from api: BotAPI) async {
        await news.load(from: api)
        await plans.load(from: api)
        await trader.restore(from: api)
    }
}
