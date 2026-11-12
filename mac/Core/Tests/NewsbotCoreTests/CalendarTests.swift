import Foundation
import Testing

@testable import NewsbotCore

@Test func decodesACalendarReminder() throws {
    let json = """
        [{"id": "today:vote:2026-11-16", "source": "calendar", "category": "reminder", "kind": "сьогодні, 16-11-2026",
          "title": "ZEC: Голосування", "time": "2026-11-15T07:00:00Z", "symbols": ["ZEC"]}]
        """

    let events = try ServerJSON.decoder.decode([BotEvent].self, from: Data(json.utf8))

    #expect(events.first?.category == .reminder && events.first?.symbols == ["ZEC"])
}

@Test func decodesAMarketEventAndCountsItsDays() throws {
    let json = """
        [{"id": "ФРС:2026-12-09T19:00:00Z:FOMC Meeting", "time": "2026-12-09T19:00:00Z", "title": "FOMC Meeting",
          "source": "ФРС", "url": "https://www.federalreserve.gov/monetarypolicy/fomccalendars.htm"},
         {"id": "coin:n2", "time": "2026-12-10T22:00:00Z", "title": "Голосування", "source": "ZEC",
          "kind": "coin", "assets": ["ZEC"], "all_day": true}]
        """
    let events = try ServerJSON.decoder.decode([MarketEvent].self, from: Data(json.utf8))
    let dayBefore = try #require(ServerJSON.parseTime("2026-12-08T10:00:00Z"))

    #expect(events.first?.source == "ФРС" && events.first?.daysLeft(from: dayBefore) == 1)
    #expect(events.first?.coin == nil && events.last?.coin == "ZEC" && events.last?.allDay == true)
    #expect(events.last?.dated == Format.day(events[1].time))

    let unlocks = """
        [{"id": "unlock:KAITO", "time": "2026-11-19T04:10:00Z", "title": "Розблокування 7.29% обігу — $3.6M",
          "source": "KAITO", "kind": "unlock", "assets": ["KAITO"], "share_of_supply": 0.0729},
         {"id": "unlock:ARB", "time": "2026-11-15T00:29:00Z", "title": "Розблокування 1.34% обігу — $13.9M",
          "source": "ARB", "kind": "unlock", "assets": ["ARB"], "share_of_supply": 0.0134},
         {"id": "vote:ZEC", "time": "2026-11-16T00:00:00Z", "title": "Голосування за NU7.1",
          "source": "ZEC", "kind": "coin", "assets": ["ZEC"], "all_day": true}]
        """
    let supply = try ServerJSON.decoder.decode([MarketEvent].self, from: Data(unlocks.utf8))

    #expect(supply[0].weighty && !supply[1].weighty && !supply[2].weighty)
    #expect(supply[2].coin == "ZEC" && supply[0].coin == "KAITO")
}

@MainActor @Test func theMainCalendarLeavesOutRoughDatesAndACoinPageShowsOnlyItsOwn() async {
    var api = FakeAPI()
    let soon = Date().addingTimeInterval(86_400)

    api.dates = [
        MarketEvent(id: "fomc", time: soon, title: "FOMC Meeting", source: "ФРС"),
        MarketEvent(
            id: "vote", time: soon, title: "Голосування", source: "ZEC", kind: "coin", assets: ["ZEC"], allDay: true),
        MarketEvent(
            id: "nu8", time: soon, title: "NU8", source: "ZEC", kind: "coin", assets: ["ZEC"], allDay: true,
            approximate: true),
        MarketEvent(id: "unlock", time: soon, title: "Розблокування", source: "AGT", kind: "coin", assets: ["AGT"]),
        MarketEvent(
            id: "passed", time: soon.addingTimeInterval(-3 * 86_400), title: "Лістинг", source: "ZEC", kind: "coin",
            assets: ["ZEC"]),
    ]

    let store = AppStore(api: api)

    await store.reloadCalendar()

    #expect(store.calendar(tradfi: true).map(\.id) == ["fomc"])
    #expect(store.calendar(tradfi: false).map(\.id) == ["vote", "unlock"])
    #expect(store.marks(of: "ZEC").map(\.id) == ["vote", "passed"] && store.marks(of: "AGT").map(\.id) == ["unlock"])
    #expect(store.dates(of: "ZEC").map(\.id) == ["vote", "nu8"])
    #expect(store.dates(of: "XAU").isEmpty && !store.traded("XAU"))
    #expect(store.dates(of: "ZEC").last?.dated.hasPrefix("≈ ") == true)
}
