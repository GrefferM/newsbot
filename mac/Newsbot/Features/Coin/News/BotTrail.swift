import NewsbotCore
import SwiftUI

struct BotTrail: View {
    let asset: String

    @Environment(AppStore.self) private var store

    var body: some View {
        Heading(title: "Сигнали й угоди бота", note: "по цьому інструменту")

        if seen.isEmpty && positions.isEmpty {
            Remark(text: "Бот ще не бачив подій про цей інструмент і не відкривав по ньому позицій.")
        }

        ForEach(seen) { event in
            TimedLine(left: Format.moment(event.time, now: Date()), right: event.kind)
        }

        ForEach(positions) { position in
            TimedLine(
                left: Format.moment(position.opened, now: Date()),
                right: position.open
                    ? "\(position.side.name) відкрито · \(Format.usd(position.stake))"
                    : "\(position.side.name) закрито · \(position.reason ?? "")"
            )
        }
    }

    private var seen: [BotEvent] {
        store.events.filter { $0.category != .trade && ($0.symbols ?? []).contains(asset) }
    }

    private var positions: [Position] {
        store.book.positions.filter { $0.symbol == asset }
    }
}
