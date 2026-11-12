import Foundation

public struct Finding: Codable, Identifiable, Equatable, Sendable {
    public let id: String
    public let asset: String
    public let published: String?
    public let day: String?
    public let text: String
    public let source: String
    public let outlet: String?
    public let others: [String]?
    public let unconfirmed: Bool
    public let approximate: Bool?

    public init(
        id: String, asset: String, published: String? = nil, day: String? = nil, text: String, source: String,
        outlet: String? = nil,
        others: [String]? = nil, unconfirmed: Bool, approximate: Bool? = nil
    ) {
        self.id = id
        self.asset = asset
        self.published = published
        self.day = day
        self.text = text
        self.source = source
        self.outlet = outlet
        self.others = others
        self.unconfirmed = unconfirmed
        self.approximate = approximate
    }

    public var sources: [String] {
        var seen: Set<String> = []

        return ([source] + (others ?? [])).filter { !$0.isEmpty && seen.insert($0).inserted }
    }

    public var stamp: String {
        published ?? day ?? ""
    }

    public func label(of source: String) -> String {
        if source == self.source, let outlet, !outlet.isEmpty {
            return outlet
        }

        let host = Self.host(of: source)
        let shared = sources.filter { Self.host(of: $0) == host }.count > 1

        guard shared, let page = URL(string: source)?.lastPathComponent, page.count > 1 else {
            return host
        }

        return host + "/…" + page.suffix(Self.pageTail)
    }

    private static let pageTail = 18

    public static func host(of source: String) -> String {
        URL(string: source)?.host()?.replacing("www.", with: "") ?? source
    }
}
