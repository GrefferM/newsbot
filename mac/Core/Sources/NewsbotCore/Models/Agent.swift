import Foundation

public enum Agent: String, CaseIterable, Codable, Sendable {
    case binance
    case osinter
    case risk
    case trader

    public var name: String {
        switch self {
        case .binance:
            "Binance-аналітик"
        case .osinter:
            "ОСІНТер"
        case .risk:
            "Ризик-менеджер"
        case .trader:
            "Трейдер"
        }
    }

    public var short: String {
        switch self {
        case .binance:
            "Binance"
        case .risk:
            "Ризик"
        default:
            name
        }
    }
}

public enum AgentError: Error, Equatable, CustomStringConvertible {
    case failed(String)

    public var description: String {
        switch self {
        case .failed(let reason):
            "Агент не відповів: \(reason)"
        }
    }
}
