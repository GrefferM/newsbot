import Foundation

public struct MarketEvent: Decodable, Identifiable, Equatable, Sendable {
    public let id: String
    public let time: Date
    public let title: String
    public let source: String
    public let url: String?
    public let kind: String?
    public let assets: [String]?
    public let allDay: Bool?
    public let approximate: Bool?
    public let share: Double?

    public init(
        id: String, time: Date, title: String, source: String, url: String? = nil, kind: String? = nil,
        assets: [String]? = nil, allDay: Bool? = nil, approximate: Bool? = nil, share: Double? = nil
    ) {
        self.id = id
        self.time = time
        self.title = title
        self.source = source
        self.url = url
        self.kind = kind
        self.assets = assets
        self.allDay = allDay
        self.approximate = approximate
        self.share = share
    }

    enum CodingKeys: String, CodingKey {
        case id, time, title, source, url, kind, assets, approximate
        case allDay = "all_day"
        case share = "share_of_supply"
    }

    public var coin: String? {
        ["coin", "unlock"].contains(kind ?? "") ? assets?.first : nil
    }

    public var weighty: Bool {
        (share ?? 0) >= 0.05
    }

    public func ahead(of now: Date) -> Bool {
        time >= now || (allDay == true && Calendar.current.isDate(time, inSameDayAs: now))
    }

    public func traded(among contracts: Set<String>) -> Bool {
        let named = assets ?? []

        return named.isEmpty || named.contains(where: contracts.contains)
    }

    public func concerns(_ asset: String, traded: Bool) -> Bool {
        let named = assets ?? []

        return named.contains(asset) || (named.isEmpty && traded)
    }

    public var dated: String {
        let day = Format.day(time)

        if approximate == true {
            return "≈ " + day
        }

        guard allDay != true else {
            return day
        }

        return day + " " + time.formatted(.dateTime.hour(.twoDigits(amPM: .omitted)).minute(.twoDigits))
    }

    public func daysLeft(from now: Date) -> Int {
        let calendar = Calendar.current

        return calendar.dateComponents([.day], from: calendar.startOfDay(for: now), to: calendar.startOfDay(for: time))
            .day ?? 0
    }
}
