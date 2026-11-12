import NewsbotCore
import SwiftUI

struct SourceList: View {
    let sources: [String: SourceHealth]

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Джерела сервера")
                .font(.system(size: 13, weight: .semibold))

            if sources.isEmpty {
                Text("Сервер ще не відповів.")
                    .font(.system(size: 12))
                    .foregroundStyle(Theme.secondary)
            }

            ForEach(sources.keys.sorted(), id: \.self) { name in
                if let source = sources[name] {
                    SourceRow(name: name, source: source)
                }
            }
        }
        .padding(16)
        .frame(width: 400, alignment: .leading)
        .foregroundStyle(Theme.text)
        .preferredColorScheme(.dark)
    }
}
