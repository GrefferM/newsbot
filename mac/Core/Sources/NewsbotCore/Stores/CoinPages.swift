import Foundation
import Observation

@MainActor
@Observable
public final class CoinPages {
    @ObservationIgnored private var pages: [String: CoinPage] = [:]

    public init() {}

    public func page(for asset: String) -> CoinPage {
        if let page = pages[asset] {
            return page
        }

        let page = CoinPage(asset: asset)
        pages[asset] = page

        return page
    }
}
