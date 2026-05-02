import Foundation

struct CLICommandOutput {
    let status: Int32
    let stdout: Data
    let stderr: Data
}

struct CLICommandRunner {
    enum RunnerError: LocalizedError {
        case binaryNotFound(String)
        case launchFailed(String)
        case invalidResponse(String)

        var errorDescription: String? {
            switch self {
            case let .binaryNotFound(message),
                 let .launchFailed(message),
                 let .invalidResponse(message):
                return message
            }
        }
    }

    let binaryURL: URL

    func run(arguments: [String]) async throws -> CLICommandOutput {
        guard FileManager.default.isExecutableFile(atPath: binaryURL.path) else {
            throw RunnerError.binaryNotFound("Missing larkfs binary at \(binaryURL.path)")
        }

        return try await withCheckedThrowingContinuation { continuation in
            let process = Process()
            let stdoutPipe = Pipe()
            let stderrPipe = Pipe()

            process.executableURL = binaryURL
            process.arguments = arguments
            process.standardOutput = stdoutPipe
            process.standardError = stderrPipe

            do {
                try process.run()
            } catch {
                continuation.resume(throwing: RunnerError.launchFailed("Failed to launch \(binaryURL.lastPathComponent): \(error.localizedDescription)"))
                return
            }

            DispatchQueue.global(qos: .userInitiated).async {
                process.waitUntilExit()
                let stdout = stdoutPipe.fileHandleForReading.readDataToEndOfFile()
                let stderr = stderrPipe.fileHandleForReading.readDataToEndOfFile()
                continuation.resume(returning: CLICommandOutput(status: process.terminationStatus, stdout: stdout, stderr: stderr))
            }
        }
    }
}
