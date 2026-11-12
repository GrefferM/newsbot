import NewsbotCore
import SwiftUI

struct SourceLink: View {
    let source: String
    let label: String

    var body: some View {
        if let url = URL(string: source), url.scheme != nil {
            Link(label, destination: url)
                .font(.system(size: 11))
                .lineLimit(1)
                .help(source)
        } else {
            Text(source)
                .font(.system(size: 11))
                .foregroundStyle(Theme.secondary)
                .lineLimit(1)
        }
    }
}
