import Foundation

public struct DeskResults: Decodable, Equatable, Sendable {
    public struct Plans: Decodable, Equatable, Sendable {
        public let armed: Int
        public let fired: Int
        public let refuted: Int
        public let expired: Int
        public let cancelled: Int

        public init(armed: Int = 0, fired: Int = 0, refuted: Int = 0, expired: Int = 0, cancelled: Int = 0) {
            self.armed = armed
            self.fired = fired
            self.refuted = refuted
            self.expired = expired
            self.cancelled = cancelled
        }

        public var total: Int {
            armed + fired + refuted + expired + cancelled
        }
    }

    public struct Trades: Decodable, Equatable, Sendable {
        public let open: Int
        public let closed: Int
        public let won: Int
        public let lost: Int
        public let pnl: Double

        public init(open: Int = 0, closed: Int = 0, won: Int = 0, lost: Int = 0, pnl: Double = 0) {
            self.open = open
            self.closed = closed
            self.won = won
            self.lost = lost
            self.pnl = pnl
        }

        enum CodingKeys: String, CodingKey {
            case open, closed, won, lost
            case pnl = "pnl_usd"
        }
    }

    public let plans: Plans
    public let trades: Trades

    public init(plans: Plans = Plans(), trades: Trades = Trades()) {
        self.plans = plans
        self.trades = trades
    }

    public var planLine: String {
        guard plans.total > 0 else {
            return "Планів на стеженні ще не було."
        }

        return "Планів \(plans.total): стежу \(plans.armed) · увійшли \(plans.fired) · відхилено \(plans.refuted) · "
            + "минув строк \(plans.expired) · знято \(plans.cancelled)"
    }

    public var tradeLine: String {
        guard trades.closed + trades.open > 0 else {
            return "Угод ще не було."
        }

        let sign = trades.pnl < 0 ? "−" : "+"
        let money = String(format: "%@$%.2f", sign, abs(trades.pnl))

        return "Угод закрито \(trades.closed): у плюс \(trades.won), у мінус \(trades.lost), разом \(money)"
            + " · відкрито \(trades.open)"
    }
}
