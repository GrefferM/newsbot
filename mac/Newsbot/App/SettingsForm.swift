import NewsbotCore
import SwiftUI

struct SettingsForm: View {
    @Binding var serverURL: String
    @Binding var tunnelServer: String
    @Binding var tunnelKey: String
    @Binding var standsFolder: String

    let failure: String?

    var body: some View {
        Form {
            TextField("Адреса сервера", text: $serverURL)

            Section("SSH-тунель — відкривається з застосунком і закривається з ним") {
                TextField("Сервер (користувач@адреса)", text: $tunnelServer)
                TextField("SSH-ключ", text: $tunnelKey, prompt: Text("~/.ssh/id_ed25519"))

                if let failure {
                    Text(failure)
                        .font(.system(size: 11))
                        .foregroundStyle(Theme.redText)
                }
            }

            Section("Стенди — локальна історія, у кожного стенду своя папка з index.html") {
                TextField("Папка стендів", text: $standsFolder, prompt: Text("~/stands"))
            }
        }
        .padding(20)
        .frame(width: 480)
    }
}
