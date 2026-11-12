import Foundation

public enum ChartInterval: String, CaseIterable, Sendable {
    case minute = "1m"
    case fiveMinutes = "5m"
    case hour = "1h"
    case day = "1d"

    public var title: String {
        switch self {
        case .minute:
            "1 хв"
        case .fiveMinutes:
            "5 хв"
        case .hour:
            "1 год"
        case .day:
            "1 день"
        }
    }

    public var seconds: TimeInterval {
        switch self {
        case .minute:
            60
        case .fiveMinutes:
            300
        case .hour:
            3600
        case .day:
            86_400
        }
    }

    public var derivativesStep: (name: String, seconds: TimeInterval) {
        switch self {
        case .minute, .fiveMinutes:
            ("5m", 300)
        case .hour:
            ("1h", 3600)
        case .day:
            ("1d", 86_400)
        }
    }
}
