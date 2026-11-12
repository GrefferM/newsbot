import NewsbotCore
import SwiftUI

struct WatchRow: View {
    let watch: PlanWatch
    let plans: CoinPlans?

    @Environment(AppStore.self) private var store
    @State private var editing = false

    private var waiting: Bool {
        plans != nil
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            row

            if let plans, editing {
                PlanEditor(plan: watch.plan) { numbers in
                    if await plans.amend(watch, to: numbers, with: store.server) {
                        editing = false
                    }
                } cancel: {
                    editing = false
                }
            }
        }
        .padding(.top, 6)
        .overlay(alignment: .top) {
            Theme.faintRule
                .frame(height: 1)
        }
    }

    private var row: some View {
        HStack(alignment: .top, spacing: 10) {
            VStack(alignment: .leading, spacing: 2) {
                Text(watch.plan.condition)
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(waiting ? Theme.text : Theme.secondary)

                if let numbers = watch.plan.numbers {
                    Text(numbers)
                        .font(.system(size: 11, design: .monospaced))
                        .foregroundStyle(Theme.caption)
                }

                if let note = watch.note, !note.isEmpty {
                    Text(note)
                        .font(.system(size: 11))
                        .foregroundStyle(watch.state == .fired ? Theme.greenText : Theme.orangeText)
                }

                if waiting {
                    Text(watch.waiting(now: Date()))
                        .font(.system(size: 11))
                        .foregroundStyle(Theme.secondary)
                }
            }
            .fixedSize(horizontal: false, vertical: true)

            Spacer(minLength: 0)

            VStack(alignment: .trailing, spacing: 6) {
                Pill(text: waiting ? "чекає" : watch.state.name, foreground: tint, background: tint.opacity(0.14))

                if let plans {
                    HStack(spacing: 6) {
                        PillButton(icon: "pencil", title: "Змінити", active: editing) {
                            editing.toggle()
                        }

                        PillButton(icon: "xmark", title: "Зняти") {
                            await plans.disarm(watch, with: store.server)
                        }
                    }
                }
            }
        }
    }

    private var tint: Color {
        switch watch.state {
        case .armed:
            Theme.blueText
        case .fired:
            Theme.greenText
        default:
            Theme.secondary
        }
    }
}
