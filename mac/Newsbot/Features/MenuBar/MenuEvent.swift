import NewsbotCore
import SwiftUI

struct MenuEvent: View {
    let event: BotEvent
    let now: Date

    var body: some View {
        HStack(alignment: .firstTextBaseline, spacing: 8) {
            Circle()
                .fill(EventLook(event: event).dot)
                .frame(width: 6, height: 6)

            Text(Format.moment(event.time, now: now))
                .font(.system(size: 12, design: .monospaced))
                .foregroundStyle(Theme.secondary)
                .frame(width: 80, alignment: .leading)

            Text(event.title)
                .font(.system(size: 13))
                .lineLimit(2)
        }
        .padding(.horizontal, 4)
        .padding(.vertical, 6)
    }
}
