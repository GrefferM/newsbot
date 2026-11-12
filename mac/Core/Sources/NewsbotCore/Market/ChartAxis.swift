import Foundation

public struct TimeMark: Identifiable, Equatable, Sendable {
    public let time: Date
    public let label: String
    public let major: Bool

    public var id: Date {
        time
    }
}

public enum ChartAxis {
    private static let day: TimeInterval = 86_400
    private static let steps: [TimeInterval] = [
        300, 900, 1800, 3600, 7200, 10_800, 21_600, 43_200, day, 2 * day, 7 * day, 14 * day, 30 * day,
    ]

    public static func ticks(from start: Date, to end: Date, zone: TimeZone = .current, most: Int = 9) -> [TimeMark] {
        let span = end.timeIntervalSince(start)

        guard span > 0, let step = steps.first(where: { span / $0 <= Double(most) }) ?? steps.last else {
            return []
        }

        let shift = TimeInterval(zone.secondsFromGMT(for: start))
        var local = ((start.timeIntervalSince1970 + shift) / step).rounded(.up) * step
        var out: [TimeMark] = []

        while local - shift <= end.timeIntervalSince1970 {
            out.append(tick(local: local, shift: shift, step: step))

            local += step
        }

        return out
    }

    private static func tick(local: TimeInterval, shift: TimeInterval, step: TimeInterval) -> TimeMark {
        let time = Date(timeIntervalSince1970: local - shift)
        let midnight = local.truncatingRemainder(dividingBy: day) == 0

        guard step < day else {
            return TimeMark(time: time, label: text(local, dayMonth), major: false)
        }

        return TimeMark(time: time, label: text(local, midnight ? dayMonth : hourMinute), major: midnight)
    }

    private static func text(_ local: TimeInterval, _ style: Date.VerbatimFormatStyle) -> String {
        Date(timeIntervalSince1970: local).formatted(style)
    }

    private static let utc = TimeZone(secondsFromGMT: 0)!
    private static let gregorian = Calendar(identifier: .gregorian)
    private static let dayMonth = Date.VerbatimFormatStyle(
        format: "\(day: .twoDigits).\(month: .twoDigits)", timeZone: utc, calendar: gregorian)
    private static let hourMinute = Date.VerbatimFormatStyle(
        format: "\(hour: .twoDigits(clock: .twentyFourHour, hourCycle: .zeroBased)):\(minute: .twoDigits)",
        timeZone: utc, calendar: gregorian)
}
