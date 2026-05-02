import Foundation

public protocol LarkFSNativeBridge: Sendable {
    func item(at path: String) async throws -> NativeBridgeItem
    func children(of path: String) async throws -> [NativeBridgeItem]
    func fetchContents(at path: String, to outputURL: URL) async throws
}

public struct LarkFSCLIFileProviderBridge: LarkFSNativeBridge {
    public enum BridgeError: LocalizedError {
        case binaryNotFound(String)
        case commandFailed(String)
        case decodeFailed(String)

        public var errorDescription: String? {
            switch self {
            case let .binaryNotFound(message),
                 let .commandFailed(message),
                 let .decodeFailed(message):
                return message
            }
        }
    }

    private let binaryURL: URL
    private let domains: String

    public init(binaryURL: URL, domains: String = "drive,wiki,im,calendar,tasks,mail,meetings") {
        self.binaryURL = binaryURL
        self.domains = domains
    }

    public func item(at path: String) async throws -> NativeBridgeItem {
        try await runJSON(["native", "item", cleanPath(path)])
    }

    public func children(of path: String) async throws -> [NativeBridgeItem] {
        try await runJSON(["native", "list", cleanPath(path)])
    }

    public func fetchContents(at path: String, to outputURL: URL) async throws {
        _ = try await run(["native", "fetch", cleanPath(path), "--output", outputURL.path])
    }

    private func runJSON<T: Decodable>(_ arguments: [String]) async throws -> T {
        let output = try await run(arguments)
        let decoder = JSONDecoder()
        do {
            return try decoder.decode(T.self, from: output.stdout)
        } catch {
            let text = String(data: output.stdout, encoding: .utf8) ?? ""
            throw BridgeError.decodeFailed("Could not decode native bridge JSON from `larkfs \(arguments.joined(separator: " "))`: \(text)")
        }
    }

    private func run(_ arguments: [String]) async throws -> NativeCommandOutput {
        guard FileManager.default.isExecutableFile(atPath: binaryURL.path) else {
            throw BridgeError.binaryNotFound("Missing larkfs binary at \(binaryURL.path)")
        }

        let fullArguments = arguments + ["--domains", domains]
        return try await withCheckedThrowingContinuation { continuation in
            let process = Process()
            let stdoutPipe = Pipe()
            let stderrPipe = Pipe()

            process.executableURL = binaryURL
            process.arguments = fullArguments
            process.standardOutput = stdoutPipe
            process.standardError = stderrPipe

            do {
                try process.run()
            } catch {
                continuation.resume(throwing: BridgeError.commandFailed("Failed to launch larkfs: \(error.localizedDescription)"))
                return
            }

            DispatchQueue.global(qos: .userInitiated).async {
                process.waitUntilExit()
                let output = NativeCommandOutput(
                    status: process.terminationStatus,
                    stdout: stdoutPipe.fileHandleForReading.readDataToEndOfFile(),
                    stderr: stderrPipe.fileHandleForReading.readDataToEndOfFile()
                )
                if output.status == 0 {
                    continuation.resume(returning: output)
                } else {
                    let stderr = String(data: output.stderr, encoding: .utf8)?
                        .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
                    continuation.resume(throwing: BridgeError.commandFailed(stderr.isEmpty ? "larkfs native bridge failed" : stderr))
                }
            }
        }
    }

    private func cleanPath(_ path: String) -> String {
        if path.isEmpty || path == "." {
            return "/"
        }
        if path.hasPrefix("/") {
            return path
        }
        return "/" + path
    }
}

struct NativeCommandOutput: Sendable {
    let status: Int32
    let stdout: Data
    let stderr: Data
}
