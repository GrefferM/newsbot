import Foundation

public struct PulseHeadline: Decodable, Identifiable, Equatable, Sendable {
    public let published: String
    public let title: String
    public let outlet: String
    public let url: String

    public var id: String {
        url + published
    }
}
