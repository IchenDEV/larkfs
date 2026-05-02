// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "LarkFSDesktop",
    platforms: [
        .macOS(.v14),
    ],
    products: [
        .executable(name: "LarkFSDesktop", targets: ["LarkFSDesktop"]),
        .library(name: "LarkFSNativeBridge", targets: ["LarkFSNativeBridge"]),
        .library(name: "LarkFSFileProviderExtension", targets: ["LarkFSFileProviderExtension"]),
    ],
    targets: [
        .target(
            name: "LarkFSNativeBridge",
            path: "NativeBridge"
        ),
        .target(
            name: "LarkFSFileProviderExtension",
            dependencies: ["LarkFSNativeBridge"],
            path: "FileProviderExtension"
        ),
        .executableTarget(
            name: "LarkFSDesktop",
            dependencies: ["LarkFSNativeBridge"],
            path: ".",
            exclude: [
                "App/Info.plist",
                "App/LarkFSDesktop.entitlements",
                "FileProviderExtension",
                "FileProviderSupport",
                "LarkFSDesktop.xcodeproj",
                "NativeBridge",
                "Resources",
            ],
            sources: [
                "App",
                "Models",
                "Services",
                "Stores",
                "Support",
                "Views",
            ]
        ),
    ]
)
