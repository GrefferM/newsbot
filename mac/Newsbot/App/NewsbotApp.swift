import NewsbotCore
import SwiftUI

@main
struct NewsbotApp: App {
    static let defaultServer = "http://127.0.0.1:8080"

    @AppStorage("serverURL") private var serverURL = NewsbotApp.defaultServer
    @AppStorage("tunnelServer") private var tunnelServer = ""
    @AppStorage("tunnelKey") private var tunnelKey = ""

    @State private var store = AppStore(api: ServerClient(base: URL(string: NewsbotApp.defaultServer)!))
    @State private var analyst = Conversation(agent: .binance)
    @State private var navigator = Navigator()
    @State private var tunnel = Tunnel()
    @State private var pages = CoinPages()

    var body: some Scene {
        Window("Newsbot", id: "main") {
            MainWindow()
                .environment(store)
                .environment(analyst)
                .environment(navigator)
        }
        .windowToolbarStyle(.unified(showsTitle: false))
        .defaultSize(width: 1400, height: 760)

        Window("Монета", id: "coin") {
            CoinView(asset: navigator.coin)
                .id(navigator.coin)
                .environment(store)
                .environment(pages)
                .environment(navigator)
        }
        .windowStyle(.hiddenTitleBar)
        .defaultSize(width: 1280, height: 900)

        WindowGroup("Позиція", id: "position", for: String.self) { $id in
            PositionWindow(id: id ?? "")
                .environment(store)
        }
        .windowStyle(.hiddenTitleBar)
        .windowResizability(.contentSize)

        MenuBarExtra {
            MenuBarView()
                .environment(store)
        } label: {
            MenuBarIcon()
                .task(id: route) {
                    await tunnel.keep(route)
                }
                .onReceive(NotificationCenter.default.publisher(for: NSApplication.willTerminateNotification)) { _ in
                    tunnel.close()
                }
                .task(id: serverURL) {
                    let server = ServerClient(base: URL(string: serverURL) ?? URL(string: NewsbotApp.defaultServer)!)

                    store.connect(to: server)
                    await analyst.restore(from: server)

                    await store.run(every: .seconds(15))
                }
        }
        .menuBarExtraStyle(.window)

        Settings {
            SettingsForm(
                serverURL: $serverURL, tunnelServer: $tunnelServer, tunnelKey: $tunnelKey, failure: tunnel.failure)
        }
    }

    private var route: TunnelRoute? {
        TunnelRoute(server: tunnelServer, key: tunnelKey, address: serverURL)
    }
}
