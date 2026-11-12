import NewsbotCore
import SwiftUI

struct ChatThread: View {
    let talk: Conversation
    var limit: CGFloat?

    @State private var height: CGFloat = 0

    var body: some View {
        ScrollViewReader { proxy in
            ScrollView {
                VStack(alignment: .leading, spacing: 12) {
                    ForEach(talk.turns) { turn in
                        TurnView(turn: turn, agent: talk.agent)
                            .id(turn.id)
                    }
                }
                .onGeometryChange(for: CGFloat.self) { proxy in
                    proxy.size.height
                } action: { content in
                    height = content
                }
            }
            .defaultScrollAnchor(.bottom)
            .frame(height: limit.map { min(height, $0) })
            .onChange(of: talk.turns.last?.answer) {
                withAnimation {
                    proxy.scrollTo(talk.turns.last?.id, anchor: .bottom)
                }
            }
        }
    }
}
