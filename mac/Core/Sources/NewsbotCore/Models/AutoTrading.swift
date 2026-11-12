import Foundation

public struct AutoTrading: Codable, Equatable, Sendable {
    public let enabled: Bool

    public init(enabled: Bool) {
        self.enabled = enabled
    }
}
