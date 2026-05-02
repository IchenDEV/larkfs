import Foundation

public enum NativeBridgeItemKind: String, Codable, Sendable {
    case directory
    case file
}

public struct NativeBridgeItem: Codable, Identifiable, Sendable {
    public let id: String
    public let parentID: String?
    public let path: String
    public let name: String
    public let kind: NativeBridgeItemKind
    public let docType: String?
    public let contentType: String
    public let size: Int64?
    public let createdAt: String?
    public let modifiedAt: String?
    public let version: String

    public enum CodingKeys: String, CodingKey {
        case id
        case parentID = "parent_id"
        case path
        case name
        case kind
        case docType = "doc_type"
        case contentType = "content_type"
        case size
        case createdAt = "created_at"
        case modifiedAt = "modified_at"
        case version
    }

    public init(
        id: String,
        parentID: String?,
        path: String,
        name: String,
        kind: NativeBridgeItemKind,
        docType: String?,
        contentType: String,
        size: Int64?,
        createdAt: String?,
        modifiedAt: String?,
        version: String
    ) {
        self.id = id
        self.parentID = parentID
        self.path = path
        self.name = name
        self.kind = kind
        self.docType = docType
        self.contentType = contentType
        self.size = size
        self.createdAt = createdAt
        self.modifiedAt = modifiedAt
        self.version = version
    }

    public var isDirectory: Bool {
        kind == .directory
    }
}

public enum NativeBridgeIdentifier {
    public static let root = "root"

    public static func path(for identifier: String) -> String? {
        if identifier == root {
            return "/"
        }
        guard identifier.hasPrefix("path.") else {
            return nil
        }
        let encoded = String(identifier.dropFirst("path.".count))
        guard let data = Data(base64URLEncoded: encoded),
              let path = String(data: data, encoding: .utf8),
              path.hasPrefix("/") else {
            return nil
        }
        return path
    }
}

extension Data {
    init?(base64URLEncoded encoded: String) {
        var base64 = encoded
            .replacingOccurrences(of: "-", with: "+")
            .replacingOccurrences(of: "_", with: "/")
        let padding = base64.count % 4
        if padding > 0 {
            base64.append(String(repeating: "=", count: 4 - padding))
        }
        self.init(base64Encoded: base64)
    }
}
