import Foundation
import Observation

@MainActor
@Observable
public final class Tunnel {
    public private(set) var failure: String?

    private var child: Child?

    private static let ssh = URL(fileURLWithPath: "/usr/bin/ssh")

    public init() {}

    public func keep(_ route: TunnelRoute?, retry: Duration = .seconds(15)) async {
        failure = nil

        guard let route else {
            return
        }

        while !Task.isCancelled {
            let reason = await run(route)

            if Task.isCancelled {
                break
            }

            failure = reason

            try? await Task.sleep(for: retry)
        }
    }

    public func close() {
        child?.stop()
        child = nil
    }

    private func run(_ route: TunnelRoute) async -> String {
        let child = Child(executable: Self.ssh, arguments: route.arguments)
        self.child = child

        return await withTaskCancellationHandler {
            await child.run()
        } onCancel: {
            child.stop()
        }
    }
}

final class Child: @unchecked Sendable {
    private let process = Process()
    private let errors = Pipe()

    init(executable: URL, arguments: [String]) {
        process.executableURL = executable
        process.arguments = arguments
        process.standardError = errors
        process.standardOutput = FileHandle.nullDevice
    }

    func run() async -> String {
        await withCheckedContinuation { done in
            process.terminationHandler = { [errors] _ in
                let said = String(decoding: errors.fileHandleForReading.availableData, as: UTF8.self)

                done.resume(returning: said.trimmingCharacters(in: .whitespacesAndNewlines))
            }

            do {
                try process.run()
            } catch {
                process.terminationHandler = nil
                done.resume(returning: error.localizedDescription)
            }
        }
    }

    func stop() {
        if process.isRunning {
            process.terminate()
        }
    }
}
