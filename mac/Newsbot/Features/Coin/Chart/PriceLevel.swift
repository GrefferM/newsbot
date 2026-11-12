import SwiftUI

struct PriceLevel: Identifiable {
    let label: String
    let price: Double
    let color: Color

    var id: String {
        label
    }
}
