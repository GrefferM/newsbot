import Foundation

public struct SourceHealth: Decodable, Equatable, Sendable {
    public let lastOK: Date?
    public let lastError: String?
    public let lastErrorAt: Date?

    public var working: Bool {
        guard let lastOK else {
            return false
        }

        guard let lastErrorAt else {
            return true
        }

        return lastOK > lastErrorAt
    }

    enum CodingKeys: String, CodingKey {
        case lastOK = "last_ok"
        case lastError = "last_error"
        case lastErrorAt = "last_error_at"
    }
}

public struct Health: Decodable, Equatable, Sendable {
    public let started: Date
    public let sources: [String: SourceHealth]
}
