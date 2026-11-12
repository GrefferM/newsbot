import Foundation

public struct TunnelRoute: Equatable, Sendable {
    public let server: String
    public let key: String
    public let port: Int

    public init?(server: String, key: String, address: String) {
        let server = server.trimmingCharacters(in: .whitespaces)

        guard !server.isEmpty, let url = URL(string: address), let host = url.host(), Self.local.contains(host) else {
            return nil
        }

        self.server = server
        self.key = key.trimmingCharacters(in: .whitespaces)
        self.port = url.port ?? Self.plainPort
    }

    public var arguments: [String] {
        var out = ["-N"]

        for option in Self.options {
            out += ["-o", option]
        }

        if !key.isEmpty {
            out += ["-i", (key as NSString).expandingTildeInPath]
        }

        return out + ["-L", "127.0.0.1:\(port):127.0.0.1:\(port)", server]
    }

    private static let local: Set<String> = ["127.0.0.1", "localhost"]

    private static let plainPort = 80

    private static let options = [
        "BatchMode=yes",
        "ExitOnForwardFailure=yes",
        "ServerAliveInterval=30",
        "ServerAliveCountMax=3",
    ]
}
