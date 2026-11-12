import Charts
import NewsbotCore
import SwiftUI

struct DerivativesCard: View {
    let contract: String
    let market: CoinMarket
    let chart: ChartModel
    let interval: ChartInterval

    @Binding var pointed: Date?

    var body: some View {
        CardFrame(title: "Деривативи", note: note) {
            if let period = chart.span, let data = market.derivatives?.within(period), !data.empty {
                let ticks = ChartAxis.ticks(from: period.lowerBound, to: period.upperBound, most: 10)

                Row(name: "Відкритий інтерес", value: interest(data), tint: Theme.text)
                SeriesPanel(
                    points: data.openInterest, color: Theme.blueText, shape: .line, period: period, ticks: ticks,
                    dated: false, value: Format.usd, pointed: $pointed)

                Row(name: "Фандинг за період", value: funding(data), tint: Theme.orangeText)
                SeriesPanel(
                    points: data.funding, color: Theme.orangeText, shape: .bars, period: period, ticks: ticks,
                    dated: false, value: Self.rate, pointed: $pointed)

                Row(name: "Лонги / шорти", value: ratio(data), tint: Theme.text)
                SeriesPanel(
                    points: data.longs, color: Theme.greenText, shape: .line, period: period, ticks: ticks,
                    dated: true, value: Self.share, pointed: $pointed)
            } else {
                Text(missing)
                    .font(.system(size: 12))
                    .foregroundStyle(Theme.secondary)
            }
        }
        .task(id: "\(contract) \(interval.rawValue) \(chart.back / (ChartModel.limit / 2))") {
            await market.followDerivatives(
                contract: contract, interval: interval,
                span: CoinMarket.reach(of: chart.span, back: chart.back, interval: interval))
        }
        .onChange(of: contract + interval.rawValue) {
            market.forgetDerivatives()
        }
    }

    private var note: String {
        if market.derivativesFailed {
            return "недоступні"
        }

        return pointed.map { "на " + Format.dayTime($0) } ?? "за вікном графіка · \(interval.title)"
    }

    private var missing: String {
        if market.derivativesFailed {
            return "Binance не віддав дані."
        }

        return market.derivatives == nil
            ? "Завантажую…" : "За цей відрізок Binance рядів не зберігає: лише останні 30 днів."
    }

    private func at(_ series: [SeriesPoint]) -> Double? {
        guard let pointed else {
            return series.last?.value
        }

        return series.nearest(to: pointed)?.value
    }

    private func interest(_ data: Derivatives) -> String {
        guard let value = at(data.openInterest) else {
            return "—"
        }

        let change = pointed == nil ? data.openInterestChange.map { " · " + Format.percent($0) } ?? "" : ""

        return Format.usd(value) + change
    }

    private func funding(_ data: Derivatives) -> String {
        at(data.funding).map(Format.funding) ?? "—"
    }

    private func ratio(_ data: Derivatives) -> String {
        guard let long = at(data.longs) else {
            return "—"
        }

        return String(format: "%.0f%% / %.0f%%", long * 100, (1 - long) * 100)
    }

    private static func rate(_ value: Double) -> String {
        Format.percent(value, digits: 2)
    }

    private static func share(_ value: Double) -> String {
        String(format: "%.0f%%", value * 100)
    }
}
