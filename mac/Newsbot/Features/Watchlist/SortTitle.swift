import SwiftUI

struct SortTitle<Key: Equatable>: View {
    let title: String
    let key: Key
    var rising = false

    @Binding var sort: Key
    @Binding var ascending: Bool

    var body: some View {
        Button {
            if sort == key {
                ascending.toggle()
            } else {
                sort = key
                ascending = rising
            }
        } label: {
            HStack(spacing: 2) {
                Text(title)
                    .lineLimit(1)
                    .fixedSize()

                if sort == key {
                    Image(systemName: ascending ? "chevron.up" : "chevron.down")
                        .font(.system(size: 8, weight: .bold))
                }
            }
            .foregroundStyle(sort == key ? Theme.text : Theme.caption)
        }
        .buttonStyle(.plain)
        .help("Сортувати")
    }
}
