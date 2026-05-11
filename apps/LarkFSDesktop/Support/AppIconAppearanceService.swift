import AppKit
import SwiftUI

enum AppIconAppearanceService {
    @MainActor
    static func apply(colorScheme: ColorScheme) {
        let iconName = colorScheme == .dark ? "AppIconDark" : "AppIcon"
        guard let image = NSImage(named: NSImage.Name(iconName)) else {
            return
        }
        NSApplication.shared.applicationIconImage = image
    }
}
