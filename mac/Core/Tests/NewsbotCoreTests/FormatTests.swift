import Foundation
import Testing

@testable import NewsbotCore

@Test(arguments: [
    "2026-11-09T18:04:41Z",
    "2026-11-09T18:04:41.891Z",
    "2026-11-09T18:04:41.891707Z",
    "2026-11-09T18:04:41.891707123Z",
])
func parsesServerTimes(text: String) {
    let date = ServerJSON.parseTime(text)

    #expect(date.map { Int($0.timeIntervalSince1970) } == 1_794_247_481)
}

@Test(arguments: [
    (1_700_000.0, "$1.70M"),
    (219_241.0, "$219K"),
    (200.0, "$200"),
    (1024.8, "$1025"),
    (84655, "$85K"),
    (-24.8, "−$25"),
])
func formatsDollars(value: Double, text: String) {
    #expect(Format.usd(value) == text)
}

@Test func anAccountIsShownToTheCent() {
    let book = Book(venue: "testnet", start: 1000, equity: 999.9654, exposure: 0, positions: [])

    #expect(book.summary == "$999.97 · −0.003%")
    #expect(Book(venue: "testnet", start: 1000, equity: 1000, exposure: 0, positions: []).summary == "$1000.00")
    #expect(Format.money(-0.0346) == "−$0.03")
    #expect(book.probe == 999.9654 * 0.2 && Book.empty.probe == 200 && book.testnet)
    #expect(book.venueName == "Testnet" && Book.empty.venueName == "не підключено")
}

@Test func formatsPercentsAndDays() {
    #expect(Format.percent(-0.68) == "−68%")
    #expect(Format.percent(1.36) == "+136%")
    #expect(Format.funding(-0.00021) == "−0.021%")
    #expect(Format.day(iso: "2026-12-09") == "09-12-2026")
    #expect(Format.stamp("2026-12-09") == "09-12-2026")
    #expect(Side.short.result(entry: 100, price: 80) == 0.25)
    #expect(abs(Side.long.result(entry: 100, price: 120) - 0.2) < 1e-9)
    #expect(Format.stamp("2026-11-10T09:30:05Z", in: TimeZone(identifier: "Europe/Kyiv")!) == "10-11-2026 11:30:05")
}

@Test func aTimeParsesWithAndWithoutFractionsAndOffsets() {
    let whole = ServerJSON.parseTime("2026-11-11T08:48:08Z")
    let fraction = ServerJSON.parseTime("2026-11-11T08:48:08.474971Z")
    let offset = ServerJSON.parseTime("2026-11-11T10:48:08+02:00")

    #expect(whole != nil && fraction != nil && offset == whole)
    #expect(fraction.map { $0.timeIntervalSince(whole ?? .distantPast) } ?? 0 > 0.4)
}

@Test func aTildeInAnAnswerIsNotReadAsStrikethrough() throws {
    let shown = try AttributedString(markdown: Format.chat("за ~6 годин, глибина ~$69 тис., **обсяг** вище"))
    let struck = shown.runs.contains { $0.inlinePresentationIntent?.contains(.strikethrough) == true }

    #expect(!struck)
    #expect(String(shown.characters) == "за ~6 годин, глибина ~$69 тис., обсяг вище")
}

@Test func aSmallPriceKeepsItsZerosSoItsWidthHolds() {
    #expect(Format.price(0.0852) == "0.085200")
    #expect(Format.price(0.07414) == "0.074140")
    #expect(Format.price(0.27233) == "0.27233")
    #expect(Format.price(0.003207) == "0.0032070")
    #expect(Format.price(3.5) == "3.5000")
    #expect(Format.price(554.7) == "554.70")
}
