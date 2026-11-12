import Foundation

public struct PulseWake: Decodable, Equatable, Sendable {
    public let minutes: Int
    public let interest: Double
    public let longs: Double

    public init(minutes: Int, interest: Double, longs: Double) {
        self.minutes = minutes
        self.interest = interest
        self.longs = longs
    }

    enum CodingKeys: String, CodingKey {
        case minutes
        case interest = "interest_5m"
        case longs = "longs_15m"
    }
}
