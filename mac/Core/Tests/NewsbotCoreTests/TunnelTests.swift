import Foundation
import Testing

@testable import NewsbotCore

@Test func aTunnelForwardsTheLocalPortOfTheServerAddress() throws {
    let route = try #require(TunnelRoute(server: " me@host ", key: "~/.ssh/key", address: "http://127.0.0.1:8080"))

    #expect(route.port == 8080 && route.server == "me@host")
    #expect(route.arguments.suffix(3) == ["-L", "127.0.0.1:8080:127.0.0.1:8080", "me@host"])
    #expect(route.arguments.contains((("~/.ssh/key") as NSString).expandingTildeInPath))
}

@Test func noTunnelWithoutAServerOrToARemoteAddress() {
    #expect(TunnelRoute(server: "", key: "", address: "http://127.0.0.1:8080") == nil)
    #expect(TunnelRoute(server: "me@host", key: "", address: "http://bot.example.org:8080") == nil)
    #expect(TunnelRoute(server: "me@host", key: "", address: "http://localhost")?.arguments.contains("-i") == false)
}

@MainActor
@Test func aTunnelThatCannotStartSaysWhyAndStopsWhenCancelled() async {
    let tunnel = Tunnel()
    let route = TunnelRoute(server: "nobody@256.0.0.1", key: "/nonexistent/key", address: "http://127.0.0.1:59999")
    let task = Task {
        await tunnel.keep(route, retry: .milliseconds(50))
    }

    try? await Task.sleep(for: .seconds(3))
    task.cancel()
    await task.value

    #expect(tunnel.failure?.isEmpty == false)
}
