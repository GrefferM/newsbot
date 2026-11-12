import NewsbotCore
import SwiftUI

struct PlanEditor: View {
    let save: (PlanNumbers) async -> Void
    let cancel: () -> Void

    @State private var numbers: PlanNumbers

    init(plan: Scenario, save: @escaping (PlanNumbers) async -> Void, cancel: @escaping () -> Void) {
        self.save = save
        self.cancel = cancel
        _numbers = State(initialValue: PlanNumbers(plan))
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 8) {
                PlanField(title: numbers.wakeTitle, text: $numbers.wake)
                PlanField(title: "Не раніше, дата й час за UTC", text: $numbers.after)
                PlanField(title: "Розмір, % рахунку", text: $numbers.size)
                PlanField(title: "Стоп", text: $numbers.stop)
                PlanField(title: "Тейк", text: $numbers.take)
            }

            HStack(spacing: 8) {
                Remark(text: "Умову плану перевіряє агент, тому вона не змінюється — лише числа.")

                Spacer()

                Button("Скасувати", action: cancel)
                    .buttonStyle(.plain)
                    .font(.system(size: 12))
                    .foregroundStyle(Theme.secondary)

                ActionButton(title: "Зберегти") {
                    await save(numbers)
                }
            }
        }
        .padding(10)
        .background(Theme.field, in: RoundedRectangle(cornerRadius: 8))
    }
}
