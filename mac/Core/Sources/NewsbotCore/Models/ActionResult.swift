import Foundation

public struct ActionResult: Decodable, Equatable, Sendable {
    public let ok: Bool
    public let error: String?
}
