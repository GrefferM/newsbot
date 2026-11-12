import NewsbotCore
import SwiftUI

struct PositionWindow: View {
    let id: String

    @Environment(AppStore.self) private var store
    @State private var live = PositionPrice(feed: BinanceFutures())

    var body: some View {
        Group {
            if let position = store.position(id) {
                content(position)
            } else {
                Text("Позиції немає")
                    .foregroundStyle(Theme.secondary)
                    .frame(width: 520, height: 200)
            }
        }
        .background(Theme.dialog)
        .foregroundStyle(Theme.text)
        .preferredColorScheme(.dark)
        .task(id: id) {
            await live.follow {
                store.position(id)
            } testnet: {
                store.book.testnet
            }
        }
    }

    private func content(_ p: Position) -> some View {
        let mark = p.open ? live.price : p.exit
        let result = mark.map { p.side.result(entry: p.entry, price: $0) } ?? p.result ?? 0

        return VStack(alignment: .leading, spacing: 0) {
            VStack(alignment: .leading, spacing: 6) {
                HStack(spacing: 8) {
                    Text(p.open ? "ПОЗИЦІЯ ВІДКРИТА" : "ПОЗИЦІЮ ЗАКРИТО · \((p.reason ?? "").uppercased())")
                        .font(.system(size: 11, weight: .bold))
                        .foregroundStyle(Theme.blueText)

                    Pill(
                        text: store.book.venueName.uppercased(), foreground: Theme.yellow,
                        background: Theme.yellow.opacity(0.18))

                    Spacer()

                    Text(Format.clock(p.opened))
                        .font(.system(size: 12, design: .monospaced))
                        .foregroundStyle(Theme.secondary)
                }

                Text("\(p.side.name) \(p.contract) · \(p.kind)")
                    .font(.system(size: 20, weight: .bold))
            }
            .padding(.horizontal, 24)
            .padding(.top, 34)
            .padding(.bottom, 16)
            .background(Theme.accent.opacity(0.10))

            HStack(alignment: .firstTextBaseline, spacing: 12) {
                Text(Format.percent(result, digits: 1))
                    .font(.system(size: 34, weight: .bold, design: .monospaced))
                    .foregroundStyle(result >= 0 ? Theme.greenText : Theme.redText)

                Text("\(Format.usd(p.stake * result)) на \(Format.usd(p.stake))")
                    .font(.system(size: 15, design: .monospaced))
                    .foregroundStyle(Theme.secondary)
            }
            .padding(.horizontal, 24)
            .padding(.top, 18)

            RangeBar(position: p, price: mark)
                .padding(.horizontal, 24)
                .padding(.top, 12)

            Grid(alignment: .leading, horizontalSpacing: 16, verticalSpacing: 10) {
                GridRow {
                    PositionLabel("Поточна ціна")
                    PositionValue(mark.map(Format.price) ?? "—")
                }
                GridRow {
                    PositionLabel("Розмір")
                    PositionValue("\(Format.percent(p.size).dropFirst()) · \(Format.usd(p.stake))")
                }
                GridRow {
                    PositionLabel("Стоп і тейк")
                    PositionValue("стоять на біржі")
                }
                GridRow {
                    PositionLabel("Закриття")
                    PositionValue(p.deadline.formatted(.dateTime.day().month().hour().minute()))
                }
            }
            .font(.system(size: 13))
            .padding(.horizontal, 16)
            .padding(.vertical, 14)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(Theme.card, in: RoundedRectangle(cornerRadius: 12))
            .padding(.horizontal, 24)
            .padding(.top, 16)

            if p.open {
                HStack(spacing: 12) {
                    Spacer()

                    Button("Закрити позицію") {
                        Task {
                            await store.close(p)
                        }
                    }
                    .buttonStyle(.plain)
                    .font(.system(size: 14, weight: .semibold))
                    .padding(.horizontal, 16)
                    .frame(height: 40)
                    .background(Theme.raised, in: RoundedRectangle(cornerRadius: 10))
                }
                .padding(.horizontal, 24)
                .padding(.top, 16)
            }

            Spacer(minLength: 20)
        }
        .frame(width: 520)
    }
}
