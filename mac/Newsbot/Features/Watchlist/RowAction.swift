import NewsbotCore
import SwiftUI

struct RowAction: View {
    let asset: String
    let browsing: Bool
    let visible: Bool

    @Environment(AppStore.self) private var store

    var body: some View {
        let kept = store.keeps(asset)

        if browsing && kept {
            Image(systemName: "checkmark.circle.fill")
                .foregroundStyle(Theme.greenText)
                .help("У вішлісті")
        } else {
            Button {
                Task {
                    if browsing {
                        await store.keep(asset, note: "")
                    } else {
                        await store.drop(asset)
                    }
                }
            } label: {
                Image(systemName: browsing ? "plus.circle.fill" : "xmark.circle.fill")
                    .foregroundStyle(browsing ? Theme.accent : Theme.caption)
            }
            .buttonStyle(.plain)
            .opacity(visible ? 1 : 0)
            .help(browsing ? "Додати до вішліста" : "Прибрати з вішліста")
            .accessibilityLabel(browsing ? "Додати \(asset)" : "Прибрати \(asset)")
        }
    }
}
