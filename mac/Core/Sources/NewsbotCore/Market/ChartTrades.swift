import Foundation

public struct ChartTrade: Identifiable, Equatable, Sendable {
    public let id: String
    public let side: Side
    public let entry: Date
    public let entryPrice: Double
    public let exit: Date
    public let exitPrice: Double
    public let result: Double
    public let lines: [String]

    public var won: Bool {
        result >= 0
    }

    public func touches(_ candle: Date?) -> Bool {
        candle == entry || candle == exit
    }
}

public enum ChartTrades {
    public static func trades(
        _ positions: [Position], contract: String, interval: ChartInterval, zone: TimeZone = .current
    ) -> [ChartTrade] {
        positions.compactMap { position in
            trade(position, contract: contract, interval: interval, zone: zone)
        }
    }

    public static func shown(_ trades: [ChartTrade], in domain: ClosedRange<Date>) -> [ChartTrade] {
        trades.filter { domain.contains($0.entry) || domain.contains($0.exit) }
    }

    private static func trade(
        _ p: Position, contract: String, interval: ChartInterval, zone: TimeZone
    ) -> ChartTrade? {
        guard p.contract == contract, !p.open, let closed = p.closedAt, let exit = p.exit, p.entry > 0, exit > 0 else {
            return nil
        }

        let result = p.result ?? p.side.result(entry: p.entry, price: exit)
        let pnl = p.pnl ?? p.stake * result
        let reason = p.reason.map { " \($0)" } ?? ""
        let lines = [
            "\(p.side.name.lowercased()) \(Format.percent(p.size).dropFirst()) · вхід \(clock(p.opened, zone)) · \(Format.price(p.entry))",
            "вихід \(clock(closed, zone))\(reason) · \(Format.price(exit))",
            "\(Format.percent(result, digits: 2)) · \(pnl >= 0 ? "+" : "")\(Format.money(pnl))",
        ]

        return ChartTrade(
            id: p.id, side: p.side,
            entry: ChartMarks.opening(of: p.opened, interval), entryPrice: p.entry,
            exit: ChartMarks.opening(of: closed, interval), exitPrice: exit,
            result: result, lines: lines
        )
    }

    private static func clock(_ date: Date, _ zone: TimeZone) -> String {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = zone

        let parts = calendar.dateComponents([.hour, .minute], from: date)

        return String(format: "%02d:%02d", parts.hour ?? 0, parts.minute ?? 0)
    }
}
