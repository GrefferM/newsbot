import Foundation

public enum EventCategory: String, Decodable, Sendable {
    case short
    case incident
    case trade
    case reminder

    public init(from decoder: Decoder) throws {
        let named = try decoder.singleValueContainer().decode(String.self)

        self = EventCategory(rawValue: named) ?? .incident
    }
}

public struct BotEvent: Decodable, Identifiable, Equatable, Sendable {
    public let id: String
    public let source: String
    public let category: EventCategory
    public let kind: String
    public let title: String
    public let url: URL?
    public let time: Date
    public let symbols: [String]?
    public let details: [String]?

    public var positionID: String? {
        id.hasPrefix("trade:open:") ? String(id.dropFirst("trade:open:".count)) : nil
    }
}
