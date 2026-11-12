import NewsbotCore
import SwiftUI

struct CalendarRow: View {
    let event: MarketEvent
    let open: () -> Void

    var body: some View {
        let days = event.daysLeft(from: Date())

        HStack(spacing: 10) {
            Text(event.dated)
                .font(.system(size: 12, design: .monospaced))
                .foregroundStyle(days <= 3 ? Theme.redText : Theme.secondary)
                .frame(width: 130, alignment: .leading)

            Text(event.source)
                .font(.system(size: 13, weight: .semibold))
                .lineLimit(1)
                .minimumScaleFactor(0.7)
                .frame(width: 56, alignment: .leading)

            Text(event.title)
                .font(.system(size: 13, weight: event.weighty ? .semibold : .regular))
                .foregroundStyle(event.weighty ? Theme.orangeText : Theme.text)
                .lineLimit(1)
                .help(event.title)

            Spacer(minLength: 4)

            Pill(
                text: days == 0 ? "сьогодні" : "за \(days) д", foreground: Theme.text.opacity(0.75),
                background: Theme.control)
        }
        .font(.system(size: 12, weight: .semibold))
        .foregroundStyle(Theme.text)
        .contentShape(Rectangle())
        .onTapGesture(perform: open)
    }
}
