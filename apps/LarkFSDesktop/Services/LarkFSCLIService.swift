import Foundation

struct LarkFSCLIService {
    enum ServiceError: LocalizedError {
        case binaryNotFound
        case decodeFailed(String)
        case commandFailed(String)

        var errorDescription: String? {
            switch self {
            case .binaryNotFound:
                return "Could not locate a runnable larkfs binary."
            case let .decodeFailed(message),
                 let .commandFailed(message):
                return message
            }
        }
    }

    func loadSnapshot() async -> DashboardLoadResult {
        guard let binaryURL = BundlePaths.larkfsBinaryURL else {
            return DashboardLoadResult(snapshot: .empty, notice: ServiceError.binaryNotFound.localizedDescription)
        }

        let runner = CLICommandRunner(binaryURL: binaryURL)

        async let versionResult: Result<VersionResponse, Error> = commandResult(["version", "--json"], with: runner)
        async let doctorResult: Result<DoctorResponse, Error> = commandResult(["doctor", "--json"], with: runner)
        async let mountsResult: Result<[MountInfo], Error> = commandResult(["status", "--json"], with: runner)

        var snapshot = DashboardSnapshot.empty
        snapshot.nativeCapability = NativeMountCapability.current()

        var notices: [String] = []

        switch await versionResult {
        case let .success(version):
            snapshot.version = version
        case let .failure(error):
            notices.append("Version: \(error.localizedDescription)")
        }

        switch await doctorResult {
        case let .success(doctor):
            snapshot.doctor = doctor
        case let .failure(error):
            notices.append("Health: \(error.localizedDescription)")
        }

        switch await mountsResult {
        case let .success(mounts):
            snapshot.mounts = mounts
        case let .failure(error):
            notices.append("Mounts: \(error.localizedDescription)")
        }

        return DashboardLoadResult(
            snapshot: snapshot,
            notice: conciseNotice(from: notices)
        )
    }

    private func commandResult<T: Decodable>(_ arguments: [String], with runner: CLICommandRunner) async -> Result<T, Error> {
        do {
            return .success(try await runJSON(arguments, with: runner))
        } catch {
            return .failure(error)
        }
    }

    private func runJSON<T: Decodable>(_ arguments: [String], with runner: CLICommandRunner) async throws -> T {
        let output = try await runner.run(arguments: arguments)
        let decoder = JSONDecoder()

        if let decoded = decodePayload(T.self, from: output.stdout, decoder: decoder) {
            return decoded
        }

        let stderr = String(data: output.stderr, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let stdout = String(data: output.stdout, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if let payload = extractJSONPayload(from: output.stdout),
           let decoded = try? decoder.decode(T.self, from: payload) {
            return decoded
        }
        if let payload = extractJSONPayload(from: output.stderr),
           let decoded = try? decoder.decode(T.self, from: payload) {
            return decoded
        }
        if output.status != 0 {
            throw ServiceError.commandFailed(stderr.isEmpty ? stdout : stderr)
        }

        throw ServiceError.decodeFailed("Could not decode JSON from `larkfs \(arguments.joined(separator: " "))`.")
    }

    private func decodePayload<T: Decodable>(_ type: T.Type, from data: Data, decoder: JSONDecoder) -> T? {
        if let decoded = try? decoder.decode(type, from: data) {
            return decoded
        }
        guard let payload = extractJSONPayload(from: data) else {
            return nil
        }
        return try? decoder.decode(type, from: payload)
    }

    private func conciseNotice(from notices: [String]) -> String? {
        let filtered = notices.filter { !$0.isEmpty }
        guard !filtered.isEmpty else { return nil }
        return filtered.joined(separator: "  ")
    }

    private func extractJSONPayload(from data: Data) -> Data? {
        guard let text = String(data: data, encoding: .utf8) else {
            return nil
        }

        guard let start = text.firstIndex(where: { $0 == "{" || $0 == "[" }) else {
            return nil
        }

        let opener = text[start]
        let closer: Character = opener == "{" ? "}" : "]"
        var depth = 0
        var inString = false
        var isEscaping = false

        for index in text[start...].indices {
            let character = text[index]

            if inString {
                if isEscaping {
                    isEscaping = false
                    continue
                }
                if character == "\\" {
                    isEscaping = true
                    continue
                }
                if character == "\"" {
                    inString = false
                }
                continue
            }

            if character == "\"" {
                inString = true
                continue
            }
            if character == opener {
                depth += 1
                continue
            }
            if character == closer {
                depth -= 1
                if depth == 0 {
                    return Data(text[start...index].utf8)
                }
            }
        }

        return nil
    }
}
