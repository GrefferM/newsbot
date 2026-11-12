import NewsbotCore
import SwiftUI

struct NewsCard: View {
    let finding: Finding
    let marked: Bool
    let action: () -> Void

    var body: some View {
        HStack(alignment: .top, spacing: 10) {
            VStack(alignment: .leading, spacing: 3) {
                Text(Format.stamp(finding.stamp))
                    .font(.system(size: 11, design: .monospaced))
                    .foregroundStyle(Theme.secondary)

                Text(finding.text)
                    .font(.system(size: 12))
                    .fixedSize(horizontal: false, vertical: true)
                    .textSelection(.enabled)

                Flow(spacing: 8) {
                    ForEach(finding.sources, id: \.self) { source in
                        SourceLink(source: source, label: finding.label(of: source))
                    }
                }
                .padding(.top, 2)
            }

            Spacer(minLength: 0)

            Button(action: action) {
                Image(systemName: "xmark.circle.fill")
                    .font(.system(size: 18))
                    .foregroundStyle(Theme.caption)
            }
            .buttonStyle(.plain)
            .help("Прибрати новину")
            .accessibilityLabel("Прибрати новину")
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 8)
        .background(Color.white.opacity(0.04), in: RoundedRectangle(cornerRadius: 9))
        .overlay(RoundedRectangle(cornerRadius: 9).stroke(marked ? Theme.accent : .clear, lineWidth: 1.5))
    }
}
