import Charts
import NewsbotCore
import SwiftUI

struct CardFrame<Content: View>: View {
    let title: String
    let note: String
    @ViewBuilder let content: Content

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text(title)
                    .font(.system(size: 12, weight: .semibold))

                Text(note)
                    .font(.system(size: 11))
                    .foregroundStyle(Theme.caption)
            }

            content
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 12)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .background(Theme.card, in: RoundedRectangle(cornerRadius: 10))
    }
}
