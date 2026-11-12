import NewsbotCore
import SwiftUI

struct ScenarioRow: View {
    let scenario: Scenario
    let plans: CoinPlans

    @Environment(AppStore.self) private var store

    var body: some View {
        HStack(alignment: .top, spacing: 10) {
            VStack(alignment: .leading, spacing: 2) {
                Text(scenario.condition)
                    .font(.system(size: 12, weight: .semibold))

                Text(scenario.action)
                    .font(.system(size: 12))

                Text(scenario.why)
                    .font(.system(size: 11))
                    .foregroundStyle(Theme.secondary)

                if let numbers = scenario.numbers {
                    Text(numbers)
                        .font(.system(size: 11, design: .monospaced))
                        .foregroundStyle(Theme.caption)
                }

                if let watch, let note = watch.note, !note.isEmpty {
                    Text(note)
                        .font(.system(size: 11))
                        .foregroundStyle(watch.state == .fired ? Theme.greenText : Theme.orangeText)
                }
            }
            .fixedSize(horizontal: false, vertical: true)

            Spacer(minLength: 0)

            VStack(alignment: .trailing, spacing: 6) {
                Pill(text: scenario.risk.rawValue, foreground: tint, background: tint.opacity(0.14))

                following
            }
        }
        .padding(.top, 6)
        .overlay(alignment: .top) {
            Theme.faintRule
                .frame(height: 1)
        }
    }

    private var watch: PlanWatch? {
        plans.watch(of: scenario)
    }

    @ViewBuilder
    private var following: some View {
        if let watch, watch.state == .armed {
            PillButton(icon: "bell.fill", title: "Стежу", active: true) {
                await plans.disarm(watch, with: store.server)
            }
            .help(watch.waiting(now: Date()) + ". Натисни, щоб зняти")
        } else if scenario.armable {
            if let watch {
                Pill(text: watch.state.name, foreground: Theme.secondary, background: .white.opacity(0.08))
            }

            PillButton(icon: "bell", title: "Стежити") {
                await plans.arm(scenario, with: store.server)
            }
            .help("Сервер стежитиме за ціною й датою плану, спитає агента, чи умова виконалась, і поставить ордер")
        }
    }

    private var tint: Color {
        switch scenario.risk {
        case .minimal:
            Theme.greenText
        case .medium:
            Theme.orangeText
        case .high:
            Theme.redText
        }
    }
}
