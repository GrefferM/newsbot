import Charts
import NewsbotCore
import SwiftUI

struct KeepButton: View {
    let asset: String

    @Environment(AppStore.self) private var store

    var body: some View {
        let kept = store.keeps(asset)

        Button(kept ? "Прибрати з вішліста" : "Додати до вішліста") {
            Task {
                if kept {
                    await store.drop(asset)
                } else {
                    await store.keep(asset, note: "")
                }
            }
        }
        .buttonStyle(.plain)
        .font(.system(size: 12, weight: .semibold))
        .padding(.horizontal, 12)
        .padding(.vertical, 6)
        .foregroundStyle(kept ? Theme.text : .white)
        .background(kept ? Theme.control : Theme.accent, in: RoundedRectangle(cornerRadius: 7))
    }
}
