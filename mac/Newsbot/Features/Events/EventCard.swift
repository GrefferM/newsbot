import NewsbotCore
import SwiftUI

struct EventCard: View {
    let event: BotEvent
    let now: Date
    let remove: () -> Void

    var body: some View {
        HStack(alignment: .top, spacing: 10) {
            Circle()
                .fill(look.dot)
                .frame(width: 8, height: 8)
                .padding(.top, 5)

            VStack(alignment: .leading, spacing: 3) {
                HStack(alignment: .firstTextBaseline) {
                    Text(look.label)
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(look.labelColor)

                    Spacer()

                    Text(Format.moment(event.time, now: now))
                        .font(.system(size: 11, design: .monospaced))
                        .foregroundStyle(Theme.caption)

                    Button(action: remove) {
                        Image(systemName: "xmark.circle.fill")
                            .font(.system(size: 14))
                            .foregroundStyle(Theme.caption)
                    }
                    .buttonStyle(.plain)
                    .help("Прибрати подію")
                    .accessibilityLabel("Прибрати подію")
                }

                Text(event.title)
                    .font(.system(size: 13))
                    .fixedSize(horizontal: false, vertical: true)

                ForEach(event.details ?? [], id: \.self) { line in
                    Text(line)
                        .font(.system(size: 12))
                        .foregroundStyle(Theme.secondary)
                        .fixedSize(horizontal: false, vertical: true)
                }
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(look.background, in: RoundedRectangle(cornerRadius: 10))
        .overlay(RoundedRectangle(cornerRadius: 10).stroke(look.border))
    }

    private var look: EventLook {
        EventLook(event: event)
    }
}
