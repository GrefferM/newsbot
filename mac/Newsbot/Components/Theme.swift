import SwiftUI

enum Theme {
    static let window = Color(hex: 0x1E1E1E)
    static let titleBar = Color(hex: 0x2A2A2C)
    static let sidebar = Color(hex: 0x252527)
    static let card = Color(hex: 0x2C2C2E)
    static let menu = Color(hex: 0x242426)
    static let rule = Color.white.opacity(0.08)
    static let faintRule = Color.white.opacity(0.05)
    static let field = Color.white.opacity(0.05)
    static let control = Color.white.opacity(0.1)
    static let raised = Color.white.opacity(0.12)
    static let dialog = Color(hex: 0x232325)

    static let text = Color.white.opacity(0.88)
    static let secondary = Color.white.opacity(0.6)
    static let caption = Color.white.opacity(0.55)

    static let accent = Color(hex: 0x0A84FF)
    static let red = Color(hex: 0xFF453A)
    static let redText = Color(hex: 0xFF6961)
    static let orange = Color(hex: 0xFF9F0A)
    static let orangeText = Color(hex: 0xFFB340)
    static let green = Color(hex: 0x30D158)
    static let greenText = Color(hex: 0x5EDB7F)
    static let yellow = Color(hex: 0xFFD60A)
    static let blueText = Color(hex: 0x64B5FF)

    static let mono = Font.system(size: 12, design: .monospaced)
}

extension Color {
    init(hex: UInt32) {
        self.init(
            red: Double((hex >> 16) & 0xFF) / 255,
            green: Double((hex >> 8) & 0xFF) / 255,
            blue: Double(hex & 0xFF) / 255
        )
    }
}
