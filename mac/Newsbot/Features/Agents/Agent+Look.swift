import NewsbotCore
import SwiftUI

extension Agent {
    var tint: Color {
        switch self {
        case .binance:
            Theme.yellow
        case .osinter:
            Theme.blueText
        case .risk:
            Theme.orangeText
        case .trader:
            Theme.greenText
        }
    }
}
