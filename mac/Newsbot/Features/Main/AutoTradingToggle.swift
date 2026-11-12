import NewsbotCore
import SwiftUI

struct AutoTradingToggle: View {
    @Environment(AppStore.self) private var store

    var body: some View {
        HStack(spacing: 8) {
            Text("Автотрейдинг")
                .font(.system(size: 12))
                .foregroundStyle(store.auto == true ? Theme.text : Theme.secondary)

            Toggle("Автотрейдинг", isOn: wanted)
                .toggleStyle(.switch)
                .controlSize(.small)
                .labelsHidden()
                .tint(Theme.green)
        }
        .disabled(store.auto == nil)
        .help(
            "За десять хвилин до події монети з годиною — розблокування чи дати з новин — ризик-менеджер "
                + "складає плани, найобережніший стає на стеження, далі вирішує суддя. Лише папір або тестнет."
        )
    }

    private var wanted: Binding<Bool> {
        Binding {
            store.auto == true
        } set: { on in
            Task {
                await store.setAuto(on)
            }
        }
    }
}
