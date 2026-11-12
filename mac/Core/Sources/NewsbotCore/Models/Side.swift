import Foundation

public enum Side: String, Codable, Sendable {
    case short
    case long

    public var name: String {
        self == .long ? "Лонг" : "Шорт"
    }

    public func result(entry: Double, price: Double) -> Double {
        self == .long ? price / entry - 1 : entry / price - 1
    }
}
