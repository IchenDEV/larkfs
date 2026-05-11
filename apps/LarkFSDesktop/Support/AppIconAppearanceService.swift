import AppKit
import SwiftUI

enum AppIconAppearanceService {
    @MainActor
    static func apply(colorScheme: ColorScheme) {
        guard let image = NSImage(named: NSImage.Name("AppIconDark")) else {
            return
        }
        NSApplication.shared.applicationIconImage = image
    }
}
