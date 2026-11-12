import NewsbotCore
import SwiftUI

struct NewsSection: View {
    let news: CoinNews
    let marked: String?

    @Environment(AppStore.self) private var store
    @State private var newestFirst = true

    var body: some View {
        HStack {
            Heading(title: "Новини", note: "знайдені ОСІНТером")

            Spacer()

            Button(newestFirst ? "Спочатку нові ↓" : "Спочатку старі ↑") {
                newestFirst.toggle()
            }
            .buttonStyle(.plain)
            .font(.system(size: 11))
            .foregroundStyle(Theme.secondary)
            .help("Порядок новин за датою публікації")
            .disabled(news.searching)
        }

        if let failure = news.failure ?? news.researcher.failure {
            Text(failure)
                .font(.system(size: 11))
                .foregroundStyle(Theme.redText)
        }

        if news.news.isEmpty && !news.searching {
            Remark(text: "Новин про інструмент ще немає. Усе, що знайде ОСІНТер, лишиться тут і піде Ризик-менеджеру.")
        }

        ForEach(news.sorted(newestFirst: newestFirst)) { finding in
            NewsCard(finding: finding, marked: marked == finding.id) {
                Task {
                    await news.forget(finding, with: store.server)
                }
            }
            .id(finding.id)
        }
    }
}
