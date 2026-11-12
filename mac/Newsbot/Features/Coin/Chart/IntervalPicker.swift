import Charts
import NewsbotCore
import SwiftUI

struct IntervalPicker: View {
    @Binding var interval: ChartInterval

    var body: some View {
        HStack(spacing: 0) {
            ForEach(ChartInterval.allCases, id: \.self) { option in
                Button(option.title) {
                    interval = option
                }
                .buttonStyle(.plain)
                .font(.system(size: 11))
                .padding(.horizontal, 9)
                .padding(.vertical, 3)
                .foregroundStyle(interval == option ? .white : Theme.secondary)
                .background(interval == option ? Color(hex: 0x636366) : .clear, in: RoundedRectangle(cornerRadius: 5))
            }
        }
        .padding(2)
        .background(Color.white.opacity(0.08), in: RoundedRectangle(cornerRadius: 7))
    }
}
