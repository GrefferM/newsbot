import NewsbotCore
import SwiftUI

struct MenuBarView: View {
    @Environment(AppStore.self) private var store
    @Environment(\.openWindow) private var openWindow

    var body: some View {
        TimelineView(.periodic(from: .now, by: 60)) { context in
            VStack(alignment: .leading, spacing: 10) {
                HStack(spacing: 8) {
                    Text("Newsbot")
                        .font(.system(size: 14, weight: .semibold))

                    Spacer()

                    ServerState()
                }
                .padding(.horizontal, 4)

                VStack(alignment: .leading, spacing: 0) {
                    Text("ОСТАННІ ПОДІЇ")
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(Theme.secondary)
                        .padding(.horizontal, 4)
                        .padding(.bottom, 6)

                    if store.events.isEmpty {
                        Text("Подій ще не було. Щойно бот щось побачить, воно з'явиться тут.")
                            .font(.system(size: 12))
                            .foregroundStyle(Theme.caption)
                            .padding(.horizontal, 4)
                            .padding(.vertical, 6)
                    }

                    ForEach(store.events.prefix(3)) { event in
                        MenuEvent(event: event, now: context.date)
                    }
                }

                Theme.rule
                    .frame(height: 1)

                VStack(spacing: 0) {
                    ForEach(store.book.openPositions) { position in
                        Button {
                            openWindow(id: "position", value: position.id)
                            NSApp.activate()
                        } label: {
                            MenuLine(
                                name: "\(position.side.name) \(position.contract)",
                                value: Format.usd(position.stake),
                                tint: Theme.blueText
                            )
                        }
                        .buttonStyle(.plain)
                    }

                    MenuLine(name: "Відкриті позиції", value: "\(store.book.openPositions.count)", tint: Theme.text)
                    MenuLine(name: "Рахунок · \(store.book.venueName)", value: store.book.summary, tint: accountTint)
                    MenuLine(name: "Вішліст", value: "\(store.watchlist.coins.count)", tint: Theme.text)
                }

                Theme.rule
                    .frame(height: 1)

                Button {
                    openWindow(id: "main")
                    NSApp.activate()
                } label: {
                    Text("Відкрити Newsbot")
                        .font(.system(size: 13, weight: .medium))
                        .foregroundStyle(.white)
                        .frame(maxWidth: .infinity)
                        .frame(height: 30)
                        .background(Theme.raised, in: RoundedRectangle(cornerRadius: 7))
                        .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
            }
            .padding(12)
            .frame(width: 340)
            .background(Theme.menu)
            .foregroundStyle(Theme.text)
            .preferredColorScheme(.dark)
        }
    }

    private var accountTint: Color {
        if store.book.change > 0 {
            return Theme.greenText
        }

        return store.book.change < 0 ? Theme.redText : Theme.text
    }
}
