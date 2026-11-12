import NewsbotCore
import SwiftUI

struct CoinAside: View {
    let asset: String
    let news: CoinNews

    @Environment(AppStore.self) private var store
    @Environment(Navigator.self) private var navigator
    @Environment(\.openURL) private var openURL
    @State private var marked: String?

    var body: some View {
        VStack(spacing: 0) {
            ScrollViewReader { proxy in
                ScrollView {
                    VStack(alignment: .leading, spacing: 10) {
                        NewsSection(news: news, marked: marked)

                        Divider().overlay(Theme.rule)

                        dates(proxy)

                        Divider().overlay(Theme.rule)

                        BotTrail(asset: asset)
                    }
                    .padding(18)
                }
                .onChange(of: navigator.news, initial: true) {
                    showAsked(with: proxy)
                }
                .onChange(of: news.news) {
                    showAsked(with: proxy)
                }
            }

            Divider().overlay(Theme.rule)

            ResearcherChat(asset: asset, news: news)
                .padding(18)
        }
        .frame(width: 440)
        .frame(maxHeight: .infinity, alignment: .top)
        .background(Theme.sidebar)
        .onChange(of: news.news, initial: true) {
            if news.loaded {
                Task {
                    await store.reloadCalendar()
                }
            }
        }
    }

    @ViewBuilder
    private func dates(_ proxy: ScrollViewProxy) -> some View {
        Heading(title: "Календар", note: "")

        let upcoming = store.dates(of: asset)

        if upcoming.isEmpty {
            Remark(text: "Найближчі події про цей інструмент з'являться тут.")
        }

        ForEach(upcoming) { event in
            CalendarRow(event: event) {
                show(event, with: proxy)
            }
        }
    }

    private func show(_ event: MarketEvent, with proxy: ScrollViewProxy) {
        guard news.news.contains(where: { $0.text == event.title }) else {
            if let address = event.url.flatMap(URL.init(string:)) {
                openURL(address)
            }

            return
        }

        navigator.show(coin: asset, news: event.title)

        showAsked(with: proxy)
    }

    private func showAsked(with proxy: ScrollViewProxy) {
        guard let text = navigator.news, let finding = news.news.first(where: { $0.text == text }) else {
            return
        }

        navigator.shown(news: text)
        marked = finding.id

        withAnimation {
            proxy.scrollTo(finding.id, anchor: .top)
        }
    }
}
