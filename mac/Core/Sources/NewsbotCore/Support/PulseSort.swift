import Foundation

public enum PulseSort: String, CaseIterable, Sendable {
    case activity
    case asset
    case quarter
    case hour
    case fourHours
    case surge
    case buyers

    public func sorted(_ coins: [PulseCoin], ascending: Bool) -> [PulseCoin] {
        coins.sorted { a, b in
            if self == .activity, a.active != b.active {
                return a.active
            }

            if self == .activity, a.rank != b.rank {
                return a.rank < b.rank
            }

            if self == .activity, !a.active, (a.trendHours ?? 0 > 0) != (b.trendHours ?? 0 > 0) {
                return a.trendHours ?? 0 > 0
            }

            let left = value(a)
            let right = value(b)

            if left == right {
                return a.asset < b.asset
            }

            return ascending ? left < right : left > right
        }
    }

    private func value(_ coin: PulseCoin) -> Double {
        switch self {
        case .activity:
            abs(coin.change1h)
        case .asset:
            0
        case .quarter:
            coin.change15m
        case .hour:
            coin.change1h
        case .fourHours:
            coin.change4h
        case .surge:
            coin.volumeSurge
        case .buyers:
            coin.buyShare
        }
    }
}
