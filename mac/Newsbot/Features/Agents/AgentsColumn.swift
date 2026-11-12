import NewsbotCore
import SwiftUI

struct AgentsColumn: View {
    @Environment(AppStore.self) private var store
    @Environment(Conversation.self) private var analyst

    @State private var draft = ""

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            SectionHeader(title: "Аналітик", note: "ринок Binance і новини") {
                ClearButton(help: "Аналітик забуде попередні питання й відповіді") {
                    await analyst.clear(with: store.server)
                }
                .disabled(analyst.working || analyst.turns.isEmpty)
            }

            Group {
                if analyst.turns.isEmpty {
                    Text(intro)
                        .font(.system(size: 13))
                        .foregroundStyle(Theme.secondary)
                        .padding(.top, 8)
                        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
                } else {
                    ChatThread(talk: analyst)
                }
            }
            .padding(.horizontal, 16)
            .padding(.bottom, 12)

            composer
        }
    }

    private var composer: some View {
        VStack(alignment: .leading, spacing: 8) {
            Flow(spacing: 6) {
                ForEach(Self.prepared, id: \.title) { phrase in
                    Hint(title: phrase.title) {
                        ask(phrase.question, reach: phrase.reach)
                    }
                    .help(phrase.help)
                }
            }
            .disabled(analyst.working)

            HStack(spacing: 8) {
                TextField("Запитай: \(analyst.agent.name)…", text: $draft)
                    .textFieldStyle(.plain)
                    .font(.system(size: 13))
                    .onSubmit(submit)

                Button(action: submit) {
                    Image(systemName: "arrow.up.circle.fill")
                        .font(.system(size: 22))
                        .foregroundStyle(Theme.accent)
                }
                .buttonStyle(.plain)
                .accessibilityLabel("Надіслати")
                .disabled(!ready)
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
            .background(Theme.card, in: RoundedRectangle(cornerRadius: 10))
            .overlay(RoundedRectangle(cornerRadius: 10).stroke(Theme.control))
        }
        .padding(.horizontal, 16)
        .padding(.top, 12)
        .padding(.bottom, 16)
        .overlay(alignment: .top) {
            Theme.rule
                .frame(height: 1)
        }
    }

    private var ready: Bool {
        !draft.trimmingCharacters(in: .whitespaces).isEmpty && !analyst.working
    }

    private static let prepared: [(title: String, question: String, reach: Reach, help: String)] = [
        (
            "Швидкі ідеї",
            "Які монети рухаються просто зараз і дають ідею короткої угоди — на хвилини чи години? Якщо таких немає — так і скажи.",
            .ideas,
            "Живий пульс ринку: рух і обіг за хвилини й години. Швидко і без платних джерел"
        ),
        (
            "Пошук кандидатів",
            "Чи є зараз монети зі зрозумілим рухом — униз чи вгору — і названою причиною? Якщо немає — так і скажи.",
            .scout,
            "Дослідник оглядає новини, далі відбір у три етапи: ринок, фіналісти, досьє на кожного. Триває довше"
        ),
        (
            "Дивності",
            "Знайди дивності на ринку: монети, чиї числа різко вибиваються з власної норми або суперечать одне одному чи новинам, і поясни кожну.",
            .ask,
            ""
        ),
    ]

    private let intro =
        "Аналітик відбирає монети за даними біржі й перевіряє, що про них пишуть. Клік по картці ідеї відкриває сторінку монети: там новини, плани входу й ордери."

    private func submit() {
        let question = draft.trimmingCharacters(in: .whitespaces)

        guard ready else {
            return
        }

        draft = ""
        ask(question)
    }

    private func ask(_ question: String, reach: Reach = .ask) {
        Task {
            await analyst.send(question, with: store.server, reach: reach)
        }
    }
}
