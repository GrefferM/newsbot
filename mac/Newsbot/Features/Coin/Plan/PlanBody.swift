import NewsbotCore
import SwiftUI

struct PlanBody: View {
    let plan: EntryPlan
    let plans: CoinPlans

    var body: some View {
        QuestionBubble(text: CoinPlans.shown(plan.wish))

        Text(LocalizedStringKey(Format.chat(plan.summary)))
            .font(.system(size: 12))
            .lineSpacing(2)
            .fixedSize(horizontal: false, vertical: true)
            .textSelection(.enabled)
            .padding(.horizontal, 10)
            .padding(.vertical, 8)
            .background(Theme.field, in: RoundedRectangle(cornerRadius: 12))
            .frame(maxWidth: .infinity, alignment: .leading)

        ForEach(plan.ranked) { scenario in
            ScenarioRow(scenario: scenario, plans: plans)
        }
    }
}
