import NewsbotCore
import SwiftUI

struct Change: View {
    let value: Double?

    var body: some View {
        Text(value.map { Format.percent($0) } ?? "—")
            .font(Theme.mono)
            .lineLimit(1)
            .foregroundStyle((value ?? 0) < 0 ? Theme.redText : Theme.greenText)
    }
}
