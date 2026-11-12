import NewsbotCore
import SwiftUI

struct EventsColumn: View {
    private var empty: String {
        switch filter {
        case .all:
            "Подій ще не було. Щойно бот побачить анонс, злам чи крок плану, воно з'явиться тут."
        case .orders:
            "Ордерів і планів на стеженні ще не було."
        case .watchlist:
            "Подій по монетах вішліста ще не було."
        case .triggers:
            "Активних тригерів немає: жоден план зараз не стоїть на стеженні."
        }
    }

    @Environment(AppStore.self) private var store
    @Environment(\.openWindow) private var openWindow
    @Environment(Navigator.self) private var navigator
    @State private var filter = EventFilter.all

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            SectionHeader(title: "Події", note: "") {
                HStack(spacing: 4) {
                    ForEach(EventFilter.allCases, id: \.self) { option in
                        Button(option.rawValue) {
                            filter = option
                        }
                        .buttonStyle(.plain)
                        .font(.system(size: 11))
                        .padding(.horizontal, 8)
                        .padding(.vertical, 3)
                        .foregroundStyle(filter == option ? Theme.text : Theme.secondary)
                        .background(filter == option ? Color.white.opacity(0.14) : .clear, in: Capsule())
                    }
                }
            }

            ScrollView {
                TimelineView(.periodic(from: .now, by: 60)) { context in
                    LazyVStack(spacing: 8) {
                        if filter == .triggers {
                            ForEach(store.triggers) { trigger in
                                TriggerCard(trigger: trigger, now: context.date) {
                                    Task {
                                        await store.unwatch(trigger)
                                    }
                                }
                                .contentShape(Rectangle())
                                .onTapGesture {
                                    navigator.show(coin: trigger.asset)
                                    openWindow(id: "coin")
                                }
                            }
                        }

                        if filter == .triggers ? store.triggers.isEmpty : store.filtered(filter).isEmpty {
                            Remark(text: empty)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(.horizontal, 4)
                                .padding(.top, 4)
                        }

                        ForEach(store.filtered(filter)) { event in
                            EventCard(event: event, now: context.date) {
                                Task {
                                    await store.forget(event)
                                }
                            }
                            .contentShape(Rectangle())
                            .onTapGesture {
                                open(event)
                            }
                        }
                    }
                    .padding(.horizontal, 12)
                    .padding(.bottom, 12)
                }
            }
        }
    }

    private func open(_ event: BotEvent) {
        if let id = event.positionID {
            openWindow(id: "position", value: id)
        } else if event.category == .trade, let asset = event.symbols?.first {
            navigator.show(coin: asset)
            openWindow(id: "coin")
        }
    }
}
