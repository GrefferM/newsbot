import Foundation

public struct WatchCoin: Decodable, Identifiable, Equatable, Sendable {
    public let asset: String
    public let note: String?
    public let added: Date?
    public let known: Bool?
    public let reasons: [String]
    public let tagged: Bool
    public let price: Double?
    public let volume: Double
    public let volumeDay: Double?
    public let change1d: Double?
    public let change7d: Double?
    public let trendVsBTC: Double?
    public let contract: String?
    public let funding: Double?
    public let openInterest: Double?

    public var id: String {
        asset
    }

    public var thin: Bool {
        reasons.contains("low-volume")
    }

    public func week(at live: Double?) -> Double? {
        guard let live, let price, let change7d, price > 0 else {
            return change7d
        }

        return live / price * (1 + change7d) - 1
    }

    public var tagName: String {
        tagged ? "Monitoring Tag" : "Без тегу"
    }

    enum CodingKeys: String, CodingKey {
        case asset, note, added, known, reasons, tagged, price, contract, funding
        case volume = "volume_30d"
        case volumeDay = "volume_day"
        case change1d = "change_1d"
        case change7d = "change_7d"
        case trendVsBTC = "trend_90d_vs_btc"
        case openInterest = "open_interest"
    }
}

public struct Watchlist: Decodable, Equatable, Sendable {
    public let built: Date?
    public let coins: [WatchCoin]

    public static let empty = Watchlist(built: nil, coins: [])
}
