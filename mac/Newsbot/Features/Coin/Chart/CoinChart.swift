import Charts
import NewsbotCore
import SwiftUI

struct CoinChart: View {
    let asset: String
    let contract: String
    let model: ChartModel

    @Binding var interval: ChartInterval
    @Binding var moment: Date?

    @Environment(AppStore.self) private var store
    @State private var pointed: Candle?
    @State private var aim: Double?
    @State private var grabbed: Int?

    var body: some View {
        let levels = levels
        let candles = model.shown
        let frame = ChartFrame(
            candles: candles, interval: interval, marks: marks, levels: levels.map(\.price), now: Date())
        let shown = pointed ?? frame.candle(at: moment, in: candles)

        VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 12) {
                if let failure = model.failure {
                    Text(failure)
                        .font(.system(size: 12))
                        .foregroundStyle(Theme.redText)
                }

                if let read = shown ?? candles.last {
                    Text(read.readout())
                        .font(.system(size: 11, design: .monospaced))
                        .foregroundStyle(read.rising ? Theme.greenText : Theme.redText)
                        .lineLimit(1)
                        .minimumScaleFactor(0.8)
                }

                Spacer()

                if model.back > 0 {
                    Button("До першої свічки") {
                        Task {
                            await model.pan(to: 0)
                        }
                    }
                    .buttonStyle(.plain)
                    .font(.system(size: 11))
                    .foregroundStyle(Theme.blueText)
                }

                IntervalPicker(interval: $interval)
            }

            Chart {
                CandleMarks(candles: candles)
                PointerMarks(shown: shown, aim: aim, last: model.latest)
                EventMarks(marks: frame.marks, shown: shown?.time)
                LevelMarks(levels: levels)
            }
            .chartXScale(domain: frame.domain)
            .chartYScale(domain: frame.prices)
            .chartYAxis {
                PriceAxis.marks(aim: aim, last: model.latest)
            }
            .chartXAxis {
                TimeAxis.marks(frame.ticks)
            }
            .chartOverlay { proxy in
                GeometryReader { area in
                    Rectangle()
                        .fill(.clear)
                        .contentShape(Rectangle())
                        .onContinuousHover { phase in
                            point(phase, proxy: proxy, area: area, prices: frame.prices)
                        }
                        .gesture(drag(across: proxy.plotSize.width))
                }
            }
            .frame(height: 240)

            VolumeChart(candles: candles, domain: frame.domain)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(Theme.card, in: RoundedRectangle(cornerRadius: 10))
    }

    private func point(_ phase: HoverPhase, proxy: ChartProxy, area: GeometryProxy, prices: ClosedRange<Double>) {
        guard case .active(let location) = phase, let frame = proxy.plotFrame else {
            pointed = nil
            aim = nil
            moment = nil

            return
        }

        let origin = area[frame].origin

        pointed = proxy.value(atX: location.x - origin.x, as: Date.self).flatMap(model.shown.nearest(to:))
        aim = proxy.value(atY: location.y - origin.y, as: Double.self).flatMap { prices.contains($0) ? $0 : nil }
        moment = pointed?.time
    }

    private func drag(across width: CGFloat) -> some Gesture {
        DragGesture(minimumDistance: 4)
            .onChanged { value in
                let start = grabbed ?? model.back
                let step = max(width, 1) / CGFloat(ChartModel.limit)

                grabbed = start

                Task {
                    await model.pan(to: start + Int((value.translation.width / step).rounded()))
                }
            }
            .onEnded { _ in
                grabbed = nil
            }
    }

    private var marks: [ChartMark] {
        let seen = store.events.filter { $0.category != .trade && ($0.symbols ?? []).contains(asset) }
        let all = ChartMarks.marks(dated: store.marks(of: asset), seen: seen, interval: interval, now: Date())

        return model.back > 0 ? all.filter { !$0.ahead } : all
    }

    private var levels: [PriceLevel] {
        guard let position = store.book.openPositions.first(where: { $0.contract == contract }) else {
            return []
        }

        return [
            PriceLevel(label: "стоп", price: position.stop, color: Theme.redText),
            PriceLevel(label: "вхід", price: position.entry, color: Theme.text),
            PriceLevel(label: "тейк", price: position.target, color: Theme.greenText),
        ]
    }
}
