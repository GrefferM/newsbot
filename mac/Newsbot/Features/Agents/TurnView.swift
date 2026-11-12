import NewsbotCore
import SwiftUI

struct TurnView: View {
    let turn: ChatTurn
    let agent: Agent

    private static let answerWidth: CGFloat = 760

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            QuestionBubble(text: turn.question)

            Text("\(agent.name) · \(Format.day(turn.at)) \(Format.clock(turn.at))")
                .font(.system(size: 11, weight: .semibold))
                .foregroundStyle(agent.tint)

            if let answer = turn.answer {
                Text(LocalizedStringKey(Format.chat(answer)))
                    .font(.system(size: 12))
                    .lineSpacing(2)
                    .fixedSize(horizontal: false, vertical: true)
                    .textSelection(.enabled)
                    .foregroundStyle(turn.failed ? Theme.redText : Theme.text)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 8)
                    .background(Theme.field, in: RoundedRectangle(cornerRadius: 12))
                    .frame(maxWidth: Self.answerWidth, alignment: .leading)
                    .frame(maxWidth: .infinity, alignment: .leading)

                ForEach(turn.ideas) { idea in
                    IdeaRow(idea: idea)
                        .frame(maxWidth: Self.answerWidth, alignment: .leading)
                }

                if !turn.calls.isEmpty {
                    Text("Інструменти: " + turn.calls.joined(separator: ", "))
                        .font(.system(size: 11))
                        .foregroundStyle(Theme.caption)
                }
            } else {
                Text("Думає, це займає кілька хвилин…")
                    .font(.system(size: 12))
                    .foregroundStyle(Theme.secondary)
            }
        }
    }
}
