import Foundation

public struct PulseCoin: Decodable, Identifiable, Equatable, Sendable {
    public let asset: String
    public let contract: String
    public let price: Double
    public let change15m: Double
    public let change1h: Double
    public let change4h: Double
    public let change24h: Double
    public let volumeHour: Double
    public let volumeSurge: Double
    public let buyShare: Double
    public let funding: Double
    public let volumeDay: Double?
    public let woke: PulseWake?
    public let launch: PulseLaunch?
    public let trendHours: Int?
    public let trendMove: Double?
    public let headlines: [PulseHeadline]

    public init(
        asset: String, contract: String = "", price: Double = 0, change15m: Double = 0, change1h: Double = 0,
        change4h: Double = 0, change24h: Double = 0, volumeHour: Double = 0, volumeSurge: Double = 0,
        buyShare: Double = 0, funding: Double = 0, volumeDay: Double? = nil, woke: PulseWake? = nil,
        launch: PulseLaunch? = nil, trendHours: Int? = nil, trendMove: Double? = nil,
        headlines: [PulseHeadline] = []
    ) {
        self.asset = asset
        self.contract = contract
        self.price = price
        self.change15m = change15m
        self.change1h = change1h
        self.change4h = change4h
        self.change24h = change24h
        self.volumeHour = volumeHour
        self.volumeSurge = volumeSurge
        self.buyShare = buyShare
        self.funding = funding
        self.volumeDay = volumeDay
        self.woke = woke
        self.launch = launch
        self.trendHours = trendHours
        self.trendMove = trendMove
        self.headlines = headlines
    }

    public var id: String {
        contract.isEmpty ? asset : contract
    }

    public var active: Bool {
        woke != nil || launch != nil
    }

    public var stage: String? {
        let parts = [launch?.told, awake].compactMap { $0 }

        return parts.isEmpty ? nil : parts.joined(separator: " · ")
    }

    public var rank: Int {
        launch?.hours ?? woke.map { 24 + $0.minutes } ?? Int.max
    }

    private var awake: String? {
        guard let woke else {
            return nil
        }

        let when = woke.minutes > 0 ? "\(woke.minutes) хв тому" : "щойно"
        let interest = Format.percent(woke.interest, digits: 1)
        let longs = Format.percent(woke.longs, digits: 1).dropLast()

        return "ожила \(when): інтерес \(interest), лонги \(longs) п.п."
    }

    public var trend: String? {
        guard let hours = trendHours, let move = trendMove, hours > 0 else {
            return nil
        }

        return "стала тенденція \(hours / 24) д: \(Format.percent(move))"
    }

    public var caption: String? {
        let parts = [stage, reason].compactMap { $0 }

        return parts.isEmpty ? nil : parts.joined(separator: " · ")
    }

    public var reason: String? {
        guard active else {
            return nil
        }

        guard let latest = headlines.first else {
            return nil
        }

        return latest.outlet.isEmpty ? latest.title : "\(latest.outlet): \(latest.title)"
    }

    enum CodingKeys: String, CodingKey {
        case asset, contract, price, woke, launch, headlines
        case trendHours = "trend_hours"
        case trendMove = "trend_move"
        case change15m = "change_15m"
        case change1h = "change_1h"
        case change4h = "change_4h"
        case change24h = "change_24h"
        case volumeHour = "volume_1h"
        case volumeDay = "volume_24h"
        case volumeSurge = "volume_surge"
        case buyShare = "buy_share"
        case funding = "funding_8h"
    }
}
