import Foundation

public struct ChartMark: Identifiable, Equatable, Sendable {
    public let time: Date
    public let labels: [String]
    public let ahead: Bool

    public var id: Date {
        time
    }

    public var label: String {
        labels.joined(separator: "  ·  ")
    }
}

public enum ChartMarks {
    public static let lead: TimeInterval = 600

    public static func marks(
        dated: [MarketEvent], seen: [BotEvent], interval: ChartInterval, now: Date, zone: TimeZone = .current
    ) -> [ChartMark] {
        var past: [Date: [String]] = [:]
        var ahead: [ChartMark] = []

        for event in dated where event.allDay != true || interval == .day {
            let text = "\(event.source): \(event.title)"

            if event.time > now {
                let left = event.time.timeIntervalSince(now)

                if left <= lead {
                    let minutes = Int((left / 60).rounded(.up))

                    ahead.append(ChartMark(time: event.time, labels: ["за \(minutes) хв", text], ahead: true))
                }

                continue
            }

            past[opening(of: event.time, interval), default: []].append(stamped(event.time, text, zone))
        }

        for event in seen where event.time <= now {
            past[opening(of: event.time, interval), default: []].append(
                stamped(event.time, "\(event.kind): \(event.title)", zone))
        }

        let behind = past.map { ChartMark(time: $0.key, labels: $0.value, ahead: false) }

        return (behind + ahead).sorted { $0.time < $1.time }
    }

    static func opening(of time: Date, _ interval: ChartInterval) -> Date {
        let seconds = interval.seconds

        return Date(timeIntervalSince1970: (time.timeIntervalSince1970 / seconds).rounded(.down) * seconds)
    }

    private static func stamped(_ time: Date, _ text: String, _ zone: TimeZone) -> String {
        Format.dayTime(time, zone: zone) + " " + text
    }
}
