import NewsbotCore
import SwiftUI

struct Heading: View {
    let title: String
    let note: String

    var body: some View {
        HStack(alignment: .firstTextBaseline, spacing: 8) {
            Text(title)
                .font(.system(size: 14, weight: .semibold))

            Text(note)
                .font(.system(size: 11))
                .foregroundStyle(Theme.caption)
        }
    }
}
