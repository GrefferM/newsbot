import NewsbotCore
import SwiftUI

struct ResearcherChat: View {
    let asset: String
    let news: CoinNews

    @Environment(AppStore.self) private var store
    @State private var query = ""

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Heading(title: "ОСІНТер", note: "розмова про \(asset)")

                Spacer()

                ClearButton(help: "ОСІНТер забуде розмову про цю монету; знайдені новини лишаться") {
                    await news.researcher.clear(with: store.server)
                }
            }

            if !news.researcher.turns.isEmpty {
                ChatThread(talk: news.researcher, limit: 300)
            }

            phrases

            search
        }
    }

    private var search: some View {
        HStack(spacing: 8) {
            TextField("Що шукати про \(asset)…", text: $query)
                .textFieldStyle(.plain)
                .font(.system(size: 12))
                .onSubmit(find)

            Button(news.searching ? "Шукаю…" : "Знайти", action: find)
                .buttonStyle(.plain)
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(.white)
                .padding(.horizontal, 10)
                .padding(.vertical, 4)
                .background(
                    news.searching ? Theme.accent.opacity(0.4) : Theme.accent, in: RoundedRectangle(cornerRadius: 6)
                )
                .disabled(news.searching)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 6)
        .background(Theme.card, in: RoundedRectangle(cornerRadius: 9))
        .overlay(RoundedRectangle(cornerRadius: 9).stroke(Theme.control))
    }

    private var phrases: some View {
        Flow(spacing: 6) {
            ForEach(store.traded(asset) ? Self.traded : Self.prepared, id: \.title) { phrase in
                Hint(title: phrase.title) {
                    query = phrase.question

                    find()
                }
            }
        }
        .disabled(news.searching)
    }

    private static let prepared: [(title: String, question: String)] = [
        ("Останні новини", CoinNews.everything),
        ("Біржі й делістинг", "Що біржі оголошували про неї: делістинги, позначки, зміни торгових пар?"),
        ("Розблокування", "Які розблокування токенів або великі продажі заплановані?"),
        ("Проєкт і команда", "Що відбувається з проєктом і командою: закриття, злами, суди, відхід людей?"),
        ("Майбутні події", "Які події з відомою датою чекають на неї попереду?"),
    ]

    private static let traded: [(title: String, question: String)] = [
        ("Останні новини", CoinNews.everything),
        (
            "Статистика й ФРС",
            "Які дані статистики США й рішення ФРС найближчим часом: дати, прогнози, попередні значення?"
        ),
        ("Звіти й прогнози", "Що зі звітами й прогнозами компанії чи галузі: дати, очікування, останні результати?"),
        ("Попит і пропозиція", "Що зараз рухає попит і пропозицію: запаси, виробники, геополітика?"),
        ("Майбутні події", "Які події з відомою датою чекають на цей інструмент попереду?"),
    ]

    private func find() {
        guard !news.searching else {
            return
        }

        let text = query

        query = ""

        Task {
            await news.search(text, with: store.server)
        }
    }
}
