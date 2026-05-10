import AppKit
import Foundation

struct TerminalCommandService {
    enum TerminalError: LocalizedError {
        case writeFailed(String)
        case openFailed(String)

        var errorDescription: String? {
            switch self {
            case let .writeFailed(message),
                 let .openFailed(message):
                return message
            }
        }
    }

    func openLarkFSInit(binaryURL: URL) throws {
        let scriptURL = BundlePaths.configDirectory.appendingPathComponent("larkfs-setup.command")
        try writeCommandScript(
            to: scriptURL,
            commandLine: "\(shellQuoted(binaryURL.path)) init",
            title: "LarkFS setup"
        )

        if !NSWorkspace.shared.open(scriptURL) {
            throw TerminalError.openFailed("Could not open Terminal for \(scriptURL.path).")
        }
    }

    private func writeCommandScript(to url: URL, commandLine: String, title: String) throws {
        do {
            try FileManager.default.createDirectory(
                at: url.deletingLastPathComponent(),
                withIntermediateDirectories: true
            )

            let script = """
            #!/bin/zsh
            clear
            echo "\(title)"
            echo ""
            echo "This window runs larkfs init so interactive Lark CLI prompts stay visible."
            echo ""
            \(commandLine)
            status=$?
            echo ""
            if [ $status -eq 0 ]; then
              echo "Setup finished. Return to LarkFS Desktop and click Refresh."
            else
              echo "Setup exited with status $status."
            fi
            echo ""
            echo "Press any key to close this window."
            read -k 1
            """

            try script.write(to: url, atomically: true, encoding: .utf8)
            try FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: url.path)
        } catch {
            throw TerminalError.writeFailed("Could not create setup script: \(error.localizedDescription)")
        }
    }

    private func shellQuoted(_ value: String) -> String {
        "'\(value.replacingOccurrences(of: "'", with: "'\\''"))'"
    }
}
