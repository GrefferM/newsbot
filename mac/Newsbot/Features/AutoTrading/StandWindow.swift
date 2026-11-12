import NewsbotCore
import SwiftUI
import UniformTypeIdentifiers
import WebKit

struct StandWindow: View {
    static let overall = "overall"

    let id: String

    @Environment(AppStore.self) private var store
    @AppStorage("standsFolder") private var folder = ""

    var body: some View {
        Group {
            if let root = StandFiles.root(folder), StandFiles.has(id, in: root) {
                StandPage(id: id, root: root, store: store)
            } else {
                Text("Стенду «\(id)» немає: вкажіть у налаштуваннях папку стендів, де лежить \(id)/index.html")
                    .font(.system(size: 12))
                    .foregroundStyle(.secondary)
                    .padding(24)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            }
        }
        .frame(minWidth: 900, minHeight: 600)
        .navigationTitle(title)
        .preferredColorScheme(.light)
    }

    private var title: String {
        if id == StandWindow.overall {
            return "Автотрейдинг разом"
        }

        return store.auto?.strategies.first { $0.id == id }?.name ?? "Стенд"
    }
}

private struct StandPage: NSViewRepresentable {
    let id: String
    let root: URL
    let store: AppStore

    func makeNSView(context: Context) -> WKWebView {
        let configuration = WKWebViewConfiguration()
        configuration.setURLSchemeHandler(StandFiles(root: root), forURLScheme: StandFiles.scheme)
        configuration.userContentController.addScriptMessageHandler(
            StandBridge(store: store, kept: root.appending(path: "\(id)/kept.json")), contentWorld: .page,
            name: StandBridge.name)

        let view = WKWebView(frame: .zero, configuration: configuration)
        view.load(URLRequest(url: URL(string: "\(StandFiles.scheme)://\(id)/index.html")!))

        return view
    }

    func updateNSView(_ view: WKWebView, context: Context) {}
}

final class StandFiles: NSObject, WKURLSchemeHandler {
    static let scheme = "stand"

    private let root: URL

    init(root: URL) {
        self.root = root.standardizedFileURL
    }

    static func root(_ folder: String) -> URL? {
        let path = NSString(string: folder).expandingTildeInPath

        return path.isEmpty ? nil : URL(filePath: path, directoryHint: .isDirectory)
    }

    static func has(_ stand: String, in root: URL) -> Bool {
        FileManager.default.fileExists(atPath: root.appending(path: "\(stand)/index.html").path)
    }

    func webView(_ webView: WKWebView, start task: any WKURLSchemeTask) {
        guard let url = task.request.url, let stand = url.host() else {
            task.didFailWithError(URLError(.badURL))

            return
        }

        let folder = root.appending(path: stand).standardizedFileURL
        let file = folder.appending(path: url.path(percentEncoded: false)).standardizedFileURL

        guard file.path.hasPrefix(folder.path + "/"), let data = try? Data(contentsOf: file) else {
            task.didReceive(HTTPURLResponse(url: url, statusCode: 404, httpVersion: nil, headerFields: nil)!)
            task.didFinish()

            return
        }

        let type = UTType(filenameExtension: file.pathExtension)?.preferredMIMEType ?? "application/octet-stream"

        task.didReceive(
            HTTPURLResponse(
                url: url, statusCode: 200, httpVersion: nil,
                headerFields: ["Content-Type": type, "Content-Length": String(data.count)])!)
        task.didReceive(data)
        task.didFinish()
    }

    func webView(_ webView: WKWebView, stop task: any WKURLSchemeTask) {}
}

@MainActor
final class StandBridge: NSObject, WKScriptMessageHandlerWithReply {
    static let name = "newsbot"

    private let store: AppStore
    private let kept: URL

    init(store: AppStore, kept: URL) {
        self.store = store
        self.kept = kept
    }

    func userContentController(
        _ controller: WKUserContentController, didReceive message: WKScriptMessage
    ) async -> (Any?, String?) {
        guard let asked = message.body as? [String: Any], let op = asked["op"] as? String else {
            return (nil, "незрозумілий запит")
        }

        switch op {
        case "rules":
            await store.refresh()

            var rules: [String: Any] = [:]

            for strategy in store.auto?.strategies ?? [] {
                rules[strategy.id] = ["name": strategy.name, "on": strategy.on, "rule": strategy.rule]
            }

            return (rules, nil)
        case "save":
            guard let id = asked["id"] as? String, let rule = asked["rule"] as? String else {
                return (nil, "незрозумілий запит")
            }

            if let error = await store.tune(id, rule: Data(rule.utf8)) {
                return (nil, error)
            }

            return (["ok": true], nil)
        case "kept":
            return ((try? String(contentsOf: kept, encoding: .utf8)) ?? "{}", nil)
        case "keep":
            guard let value = asked["value"] as? String else {
                return (nil, "незрозумілий запит")
            }

            do {
                try Data(value.utf8).write(to: kept, options: .atomic)
            } catch {
                return (nil, error.localizedDescription)
            }

            return (["ok": true], nil)
        default:
            return (nil, "незрозумілий запит")
        }
    }
}
