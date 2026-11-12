import Foundation

public enum Format {
    public static func usd(_ value: Double) -> String {
        let magnitude = abs(value)
        let sign = value < 0 ? "−" : ""

        switch magnitude {
        case 1e9...:
            return sign + "$" + String(format: "%.2fB", magnitude / 1e9)
        case 1e6...:
            return sign + "$" + String(format: "%.2fM", magnitude / 1e6)
        case 1e4...:
            return sign + "$" + String(format: "%.0fK", magnitude / 1e3)
        default:
            return sign + "$" + String(format: "%.0f", magnitude)
        }
    }

    public static func money(_ value: Double) -> String {
        (value < 0 ? "−" : "") + String(format: "$%.2f", abs(value))
    }

    public static func percent(_ fraction: Double, digits: Int = 0) -> String {
        let text = String(format: "%.\(digits)f%%", abs(fraction) * 100)

        return (fraction < 0 ? "−" : "+") + text
    }

    public static func funding(_ rate: Double) -> String {
        percent(rate, digits: 3)
    }

    public static func price(_ value: Double) -> String {
        switch value {
        case 100...:
            String(format: "%.2f", value)
        case 1...:
            String(format: "%.4f", value)
        case ..<Double.leastNonzeroMagnitude:
            "0"
        default:
            String(format: "%.*f", Int(-log10(value).rounded(.down)) + 4, value)
        }
    }

    public static func chat(_ text: String) -> String {
        text.replacingOccurrences(of: "~", with: "\\~")
    }

    public static func clock(_ date: Date) -> String {
        date.formatted(.dateTime.hour(.twoDigits(amPM: .omitted)).minute(.twoDigits).second(.twoDigits))
    }

    public static func moment(_ date: Date, now: Date) -> String {
        if Calendar.current.isDate(date, inSameDayAs: now) {
            return clock(date)
        }

        return day(date)
    }

    public static func dayTime(_ date: Date, zone: TimeZone = .current) -> String {
        date.formatted(
            Date.VerbatimFormatStyle(
                format: "\(day: .twoDigits).\(month: .twoDigits) \(hour: hour):\(minute: .twoDigits)",
                timeZone: zone, calendar: gregorian))
    }

    public static func day(_ date: Date) -> String {
        dayFormat.string(from: date)
    }

    public static func day(iso: String) -> String {
        let parts = iso.split(separator: "-")

        guard parts.count == 3 else {
            return iso
        }

        return parts.reversed().joined(separator: "-")
    }

    public static func stamp(_ published: String, in zone: TimeZone = .current) -> String {
        guard published.contains("T"), let moment = ServerJSON.parseTime(published) else {
            return day(iso: published)
        }

        let day = Date.VerbatimFormatStyle(
            format: "\(day: .twoDigits)-\(month: .twoDigits)-\(year: .defaultDigits)", timeZone: zone,
            calendar: gregorian)
        let time = Date.VerbatimFormatStyle(
            format: "\(hour: hour):\(minute: .twoDigits):\(second: .twoDigits)", timeZone: zone, calendar: gregorian)

        return moment.formatted(day) + " " + moment.formatted(time)
    }

    private static let gregorian = Calendar(identifier: .gregorian)
    private static let hour = Date.FormatStyle.Symbol.VerbatimHour.twoDigits(
        clock: .twentyFourHour, hourCycle: .zeroBased)

    private static let dayFormat: DateFormatter = {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.dateFormat = "dd-MM-yyyy"

        return formatter
    }()
}
