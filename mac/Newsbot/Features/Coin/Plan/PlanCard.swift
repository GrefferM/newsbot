import NewsbotCore
import SwiftUI

struct PlanCard: View {
    let plans: CoinPlans

    @Environment(AppStore.self) private var store

    var body: some View {
        CardFrame(title: "Плани входу", note: note) {
            if let asking = plans.asking {
                QuestionBubble(text: asking)

                Text("Ризик-менеджер складає плани, це займає кілька хвилин…")
                    .font(.system(size: 12))
                    .foregroundStyle(Theme.secondary)
            } else if let plan = plans.plan {
                PlanBody(plan: plan, plans: plans)
            } else {
                Remark(
                    text:
                        "Ризик-менеджер складе кілька планів входу — від найризикованішого до найобережнішого — за карткою монети й усіма знайденими новинами."
                )
            }

            if let failure = plans.failure {
                Text(failure)
                    .font(.system(size: 11))
                    .foregroundStyle(Theme.redText)
            }

            AskField(
                prompt: "Сформувати плани для \(plans.asset)…", title: title, busy: plans.planning
            ) { wish in
                await plans.make(wish, with: store.server)
            }
        }
        .fixedSize(horizontal: false, vertical: true)
    }

    private var title: String {
        if plans.planning {
            return "Складаю плани…"
        }

        return plans.plan == nil ? "Скласти плани" : "Оновити плани"
    }

    private var note: String {
        guard let plan = plans.plan else {
            return "Ризик-менеджер"
        }

        return "Ризик-менеджер · \(Format.day(plan.made)) \(Format.clock(plan.made))"
    }
}
