import NewsbotCore
import SwiftUI

struct TriggerCard: View {
    let trigger: PlanWatch
    let now: Date
    let remove: () -> Void

    var body: some View {
        HStack(alignment: .top, spacing: 10) {
            Circle()
                .fill(Theme.accent)
                .frame(width: 8, height: 8)
                .padding(.top, 5)

            VStack(alignment: .leading, spacing: 3) {
                HStack(alignment: .firstTextBaseline) {
                    Text(label)
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(Theme.blueText)

                    Spacer()

                    Text(Format.moment(trigger.armed, now: now))
                        .font(.system(size: 11, design: .monospaced))
                        .foregroundStyle(Theme.caption)

                    Button(action: remove) {
                        Image(systemName: "xmark.circle.fill")
                            .font(.system(size: 14))
                            .foregroundStyle(Theme.caption)
                    }
                    .buttonStyle(.plain)
                    .help("Зняти план зі стеження")
                    .accessibilityLabel("Зняти план зі стеження")
                }

                Text(trigger.plan.condition)
                    .font(.system(size: 13))
                    .fixedSize(horizontal: false, vertical: true)

                ForEach(lines, id: \.self) { line in
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
        .background(Color.white.opacity(0.04), in: RoundedRectangle(cornerRadius: 10))
        .overlay(RoundedRectangle(cornerRadius: 10).stroke(Color.white.opacity(0.08)))
    }

    private var label: String {
        ["ТРИГЕР", trigger.asset, trigger.plan.direction?.uppercased()].compactMap { $0 }.joined(separator: " · ")
    }

    private var lines: [String] {
        [trigger.plan.action, trigger.plan.numbers, trigger.waiting(now: now), trigger.note]
            .compactMap { $0 }
            .filter { !$0.isEmpty }
    }
}
