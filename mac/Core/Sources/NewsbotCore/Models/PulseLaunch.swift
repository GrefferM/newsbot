import Foundation

public struct PulseLaunch: Decodable, Equatable, Sendable {
    public let hours: Int
    public let move: Double
    public let times: Double

    public init(hours: Int, move: Double, times: Double) {
        self.hours = hours
        self.move = move
        self.times = times
    }

    public var told: String {
        let when = hours > 0 ? "\(hours) год тому" : "зараз"

        return "зліт із тиші \(when): \(Format.percent(move)) за годину, ×\(Int(times.rounded())) звичайної"
    }
}
