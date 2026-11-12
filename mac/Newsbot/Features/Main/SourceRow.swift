import NewsbotCore
import SwiftUI

struct SourceRow: View {
    let name: String
    let source: SourceHealth

    var body: some View {
        HStack(alignment: .firstTextBaseline, spacing: 8) {
            Circle()
                .fill(source.working ? Theme.green : Theme.red)
                .frame(width: 7, height: 7)

            VStack(alignment: .leading, spacing: 2) {
                Text(Self.titles[name] ?? name)
                    .font(.system(size: 12, weight: .medium))

                if !source.working, let error = source.lastError {
                    Text(error)
                        .font(.system(size: 11))
                        .foregroundStyle(Theme.redText)
                        .lineLimit(2)
                        .textSelection(.enabled)
                }
            }

            Spacer(minLength: 8)

            Text(checked)
                .font(.system(size: 11, design: .monospaced))
                .foregroundStyle(Theme.caption)
        }
    }

    private var checked: String {
        guard let lastOK = source.lastOK else {
            return "ще не відповідало"
        }

        return Format.moment(lastOK, now: Date())
    }

    private static let titles = [
        "binance-stream": "Binance · потік анонсів у реальному часі",
        "pulse": "Binance · пульс ринку щохвилини",
        "market-calendar": "Календар · ФРС, статистика США, звіти, розблокування, дати монет",
        "market-reminders": "Календар · нагадування за 10 хв і на початку події",
        "defillama-hacks": "DefiLlama · злами",
        "stablecoin-depeg": "DefiLlama · відв'язка стейблкоїнів",
        "watchlist": "Binance · скан монет для вішліста",
    ]
}
