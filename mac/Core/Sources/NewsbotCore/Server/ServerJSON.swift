import Foundation

public enum ServerJSON {
    public static let decoder: JSONDecoder = {
        let decoder = JSONDecoder()

        decoder.dateDecodingStrategy = .custom { d in
            let text = try d.singleValueContainer().decode(String.self)

            guard let date = parseTime(text) else {
                throw DecodingError.dataCorrupted(.init(codingPath: d.codingPath, debugDescription: text))
            }

            return date
        }

        return decoder
    }()

    public static func parseTime(_ text: String) -> Date? {
        let trimmed = text.replacing(/(\.\d{3})\d+/) { $0.output.1 }

        return (try? Date(trimmed, strategy: fractional)) ?? (try? Date(trimmed, strategy: whole))
    }

    private static let fractional = Date.ISO8601FormatStyle(includingFractionalSeconds: true)

    private static let whole = Date.ISO8601FormatStyle()
}
