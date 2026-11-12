import NewsbotCore
import SwiftUI

struct CoinView: View {
    let asset: String

    @Environment(AppStore.self) private var store
    @Environment(CoinPages.self) private var pages
    @State private var interval = ChartInterval.hour
    @State private var moment: Date?

    private var page: CoinPage {
        pages.page(for: asset)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            CoinHeader(asset: asset, coin: coin, live: live, chart: page.chart)

            HStack(alignment: .top, spacing: 0) {
                ScrollView {
                    VStack(alignment: .leading, spacing: 12) {
                        CoinTiles(coin: coin, live: live)

                        if let live {
                            PulseTiles(pulsed: live)
                        }

                        CoinChart(
                            asset: asset, contract: contract, model: page.chart, interval: $interval, moment: $moment)

                        HStack(alignment: .top, spacing: 12) {
                            LiquidityCard(contract: contract, market: page.market)
                            DerivativesCard(
                                contract: contract, market: page.market, chart: page.chart, interval: interval,
                                pointed: $moment)
                        }
                        .frame(height: 340)

                        PlanCard(plans: page.plans)
                        TraderCard(talk: page.trader)
                        OrdersCard(asset: asset, plans: page.plans)
                    }
                    .padding(.horizontal, 24)
                    .padding(.vertical, 16)
                }

                CoinAside(asset: asset, news: page.news)
            }
        }
        .background(Theme.window)
        .foregroundStyle(Theme.text)
        .preferredColorScheme(.dark)
        .ignoresSafeArea(.container, edges: .top)
        .frame(minWidth: 1100, minHeight: 760)
        .task(id: "\(asset) \(store.connections)") {
            await page.load(from: store.server)
        }
        .onChange(of: store.pulse(of: asset)?.headlines) {
            Task {
                await page.news.load(from: store.server)
            }
        }
        .onChange(of: store.events.first?.id) {
            Task {
                await page.plans.reloadWatches(from: store.server)
            }
        }
        .task(id: contract + interval.rawValue) {
            await page.chart.follow(contract: contract, interval: interval)
        }
    }

    private var coin: WatchCoin? {
        store.coin(asset)
    }

    private var contract: String {
        coin?.contract ?? asset + "USDT"
    }

    private var live: PulseCoin? {
        store.pulse(of: asset)
    }
}
