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
        try removeLegacyCommandScript()

        let scriptURL = BundlePaths.configDirectory.appendingPathComponent("larkfs-setup.zsh")
        try writeShellScript(
            to: scriptURL,
            commandLine: "\(shellQuoted(binaryURL.path)) init",
            title: "LarkFS setup"
        )

        try runInTerminal(command: "/bin/zsh \(shellQuoted(scriptURL.path))")
    }

    private func writeShellScript(to url: URL, commandLine: String, title: String) throws {
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

    private func runInTerminal(command: String) throws {
        let source = """
        tell application "Terminal"
            activate
            do script \(appleScriptString(command))
        end tell
        """

        guard let script = NSAppleScript(source: source) else {
            throw TerminalError.openFailed("Could not create Terminal automation script.")
        }

        var errorInfo: NSDictionary?
        script.executeAndReturnError(&errorInfo)
        if let errorInfo {
            let message = errorInfo[NSAppleScript.errorMessage] as? String
            throw TerminalError.openFailed(message ?? "Could not open Terminal. Allow Automation access for LarkFSDesktop and try again.")
        }
    }

    private func removeLegacyCommandScript() throws {
        let legacyURL = BundlePaths.configDirectory.appendingPathComponent("larkfs-setup.command")
        if FileManager.default.fileExists(atPath: legacyURL.path) {
            do {
                try FileManager.default.removeItem(at: legacyURL)
            } catch {
                throw TerminalError.writeFailed("Could not remove old setup command: \(error.localizedDescription)")
            }
        }
    }

    private func shellQuoted(_ value: String) -> String {
        "'\(value.replacingOccurrences(of: "'", with: "'\\''"))'"
    }

    private func appleScriptString(_ value: String) -> String {
        let escaped = value
            .replacingOccurrences(of: "\\", with: "\\\\")
            .replacingOccurrences(of: "\"", with: "\\\"")
        return "\"\(escaped)\""
    }
}
