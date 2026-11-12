import SwiftUI

struct SectionHeader<Trailing: View>: View {
    let title: String
    let note: String
    @ViewBuilder let trailing: Trailing

    var body: some View {
        HStack(alignment: .firstTextBaseline, spacing: 8) {
            Text(title)
                .font(.system(size: 15, weight: .semibold))

            Text(note)
                .font(.system(size: 12))
                .foregroundStyle(Theme.caption)

            Spacer()

            trailing
        }
        .padding(.horizontal, 16)
        .padding(.top, 14)
        .padding(.bottom, 10)
    }
}
