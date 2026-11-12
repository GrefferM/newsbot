import NewsbotCore
import SwiftUI

struct EventLook {
    let label: String
    let labelColor: Color
    let dot: Color
    let background: Color
    let border: Color

    init(event: BotEvent) {
        let quiet = Color.white.opacity(0.04)
        let edge = Color.white.opacity(0.08)

        switch event.category {
        case .trade:
            self.init("ОРДЕР · \(event.kind.uppercased())", Theme.blueText, Theme.accent, quiet, edge)
        case .short:
            self.init(event.kind.uppercased(), Theme.redText, Theme.red, quiet, edge)
        case .reminder:
            self.init("КАЛЕНДАР · \(event.kind.uppercased())", Theme.orangeText, Theme.orange, quiet, edge)
        case .incident:
            self.init(event.kind.uppercased(), Theme.secondary, .white.opacity(0.4), quiet, edge)
        }
    }

    private init(_ label: String, _ labelColor: Color, _ dot: Color, _ background: Color, _ border: Color) {
        self.label = label
        self.labelColor = labelColor
        self.dot = dot
        self.background = background
        self.border = border
    }
}
