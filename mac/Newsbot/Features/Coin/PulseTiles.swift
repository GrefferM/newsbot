import NewsbotCore
import SwiftUI

struct PulseTiles: View {
    let pulsed: PulseCoin

    var body: some View {
        HStack(spacing: 12) {
            move("15 хв", pulsed.change15m)
            move("1 год", pulsed.change1h)
            move("4 год", pulsed.change4h)
            move("Доба", pulsed.change24h)
            Tile(
                label: "Обіг години",
                value: String(format: "×%.1f", pulsed.volumeSurge),
                note: "\(Format.usd(pulsed.volumeHour)) за годину",
                tint: pulsed.volumeSurge >= 2 ? Theme.orangeText : Theme.text
            )
            Tile(
                label: "Покупці",
                value: String(Format.percent(pulsed.buyShare).dropFirst()),
                note: "ринкові покупки",
                tint: pulsed.buyShare >= 0.5 ? Theme.greenText : Theme.redText
            )
        }
    }

    private func move(_ label: String, _ change: Double) -> Tile {
        Tile(
            label: label, value: Format.percent(change, digits: 1), note: "рух ціни",
            tint: change < 0 ? Theme.redText : Theme.greenText)
    }
}
