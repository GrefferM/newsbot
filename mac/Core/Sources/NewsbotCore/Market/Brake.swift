import Foundation

public final class Brake: @unchecked Sendable {
    public static let shared = Brake()

    private static let shortest: TimeInterval = 60

    private let lock = NSLock()
    private var held: [String: Date] = [:]

    public init() {}

    public func holds(_ host: String, now: Date = Date()) -> Bool {
        lock.withLock {
            held[host].map { now < $0 } ?? false
        }
    }

    public func hold(_ host: String, asked seconds: TimeInterval?, now: Date = Date()) {
        lock.withLock {
            held[host] = now.addingTimeInterval(max(seconds ?? 0, Self.shortest))
        }
    }
}
