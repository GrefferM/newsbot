import Foundation

public struct PulseBoard: Decodable, Equatable, Sendable {
    public let at: Date?
    public let coins: [PulseCoin]

    public init(at: Date? = nil, coins: [PulseCoin] = []) {
        self.at = at
        self.coins = coins
    }

    public var movers: Int {
        coins.filter(\.active).count
    }

    public func found(_ text: String) -> [PulseCoin] {
        let query = text.trimmingCharacters(in: .whitespaces).uppercased()

        return query.isEmpty ? coins : coins.filter { $0.asset.uppercased().contains(query) }
    }

    public static let empty = PulseBoard()
}
