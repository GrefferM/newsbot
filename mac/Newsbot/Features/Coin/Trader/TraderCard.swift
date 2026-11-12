import NewsbotCore
import SwiftUI

struct TraderCard: View {
    let talk: Conversation

    @Environment(AppStore.self) private var store

    var body: some View {
        CardFrame(title: "Трейдер", note: "розмова про \(talk.topic)") {
            if talk.turns.isEmpty {
                Remark(
                    text:
                        "Трейдер відповідає на питання про монету в довільній формі: бачить ринок, рахунок, знайдені новини, плани й події бота. Планів він не складає — це робить Ризик-менеджер вище."
                )
            } else {
                ChatThread(talk: talk, limit: 640)
            }

            HStack(spacing: 8) {
                AskField(
                    prompt: "Спитай Трейдера про \(talk.topic)…", title: talk.working ? "Думає…" : "Запитати",
                    busy: talk.working
                ) { text in
                    await ask(text)
                }

                ClearButton(help: "Трейдер забуде розмову про цю монету") {
                    await talk.clear(with: store.server)
                }
            }
        }
        .fixedSize(horizontal: false, vertical: true)
    }

    private func ask(_ text: String) async {
        let question = text.trimmingCharacters(in: .whitespaces)

        guard !question.isEmpty else {
            return
        }

        await talk.send(question, with: store.server)
    }
}
