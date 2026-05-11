#!/usr/bin/env bash
set -euo pipefail

MODE="${1:-run}"
APP_NAME="LarkFSDesktop"
BUNDLE_ID="dev.ichen.larkfs.desktop"
MIN_SYSTEM_VERSION="14.0"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_DIR="$ROOT_DIR/apps/LarkFSDesktop"
DIST_DIR="$ROOT_DIR/dist"
XCODE_PROJECT="$APP_DIR/LarkFSDesktop.xcodeproj"
XCODE_DERIVED_DATA="$DIST_DIR/xcode-derived"
XCODE_APP_BUNDLE="$XCODE_DERIVED_DATA/Build/Products/Debug/$APP_NAME.app"
XCODE_DESTINATION="platform=macOS,arch=$(uname -m)"
SWIFTPM_APP_BUNDLE="$DIST_DIR/$APP_NAME.app"
APP_BUNDLE="$SWIFTPM_APP_BUNDLE"
APP_CONTENTS="$APP_BUNDLE/Contents"
APP_MACOS="$APP_CONTENTS/MacOS"
APP_RESOURCES="$APP_CONTENTS/Resources"
APP_RESOURCES_BIN="$APP_RESOURCES/bin"
APP_BINARY="$APP_MACOS/$APP_NAME"
INFO_PLIST="$APP_CONTENTS/Info.plist"

pkill -x "$APP_NAME" >/dev/null 2>&1 || true

build_bridge() {
  mkdir -p "$ROOT_DIR/bin"

  if ! (
    cd "$ROOT_DIR"
    go build -o "$ROOT_DIR/bin/larkfs" ./cmd/larkfs
  ); then
    if [[ ! -x "$ROOT_DIR/bin/larkfs" ]]; then
      echo "failed to build larkfs and no existing binary is available" >&2
      exit 1
    fi
    echo "warning: failed to rebuild larkfs; using existing bin/larkfs" >&2
  fi
}

build_xcode_app() {
  build_args=(
    -project "$XCODE_PROJECT" \
    -scheme "$APP_NAME" \
    -configuration Debug \
    -derivedDataPath "$XCODE_DERIVED_DATA" \
    -destination "$XCODE_DESTINATION" \
    -quiet
  )

  if [[ -n "${LARKFS_DEVELOPMENT_TEAM:-}" ]]; then
    build_args+=(
      -allowProvisioningUpdates
      DEVELOPMENT_TEAM="$LARKFS_DEVELOPMENT_TEAM"
      CODE_SIGN_STYLE=Automatic
      CODE_SIGNING_ALLOWED=YES
    )
  else
    build_args+=(CODE_SIGNING_ALLOWED=NO)
  fi

  xcodebuild "${build_args[@]}" \
    build

  APP_BUNDLE="$XCODE_APP_BUNDLE"
  APP_CONTENTS="$APP_BUNDLE/Contents"
  APP_MACOS="$APP_CONTENTS/MacOS"
  APP_RESOURCES="$APP_CONTENTS/Resources"
  APP_RESOURCES_BIN="$APP_RESOURCES/bin"
  APP_BINARY="$APP_MACOS/$APP_NAME"
  INFO_PLIST="$APP_CONTENTS/Info.plist"
}

build_swiftpm_app() {
  build_bridge

  swift build --package-path "$APP_DIR"
  BUILD_BINARY="$(swift build --package-path "$APP_DIR" --show-bin-path)/$APP_NAME"

  rm -rf "$APP_BUNDLE"
  mkdir -p "$APP_MACOS"
  mkdir -p "$APP_RESOURCES_BIN"
  cp "$BUILD_BINARY" "$APP_BINARY"
  cp "$ROOT_DIR/bin/larkfs" "$APP_RESOURCES_BIN/larkfs"
  find "$APP_DIR/Resources" -maxdepth 1 -type f \( -name 'AppIcon*.icns' -o -name 'AppIcon*.png' \) -exec cp {} "$APP_RESOURCES" \;
  chmod +x "$APP_BINARY"
  chmod +x "$APP_RESOURCES_BIN/larkfs"

  cat >"$INFO_PLIST" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleIconFile</key>
  <string>AppIconDark</string>
  <key>CFBundleExecutable</key>
  <string>$APP_NAME</string>
  <key>CFBundleIdentifier</key>
  <string>$BUNDLE_ID</string>
  <key>CFBundleName</key>
  <string>$APP_NAME</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>LSMinimumSystemVersion</key>
  <string>$MIN_SYSTEM_VERSION</string>
  <key>NSPrincipalClass</key>
  <string>NSApplication</string>
  <key>LarkFSWorkspaceRoot</key>
  <string>$ROOT_DIR</string>
</dict>
</plist>
PLIST
}

if [[ -d "$XCODE_PROJECT" ]]; then
  build_xcode_app
else
  build_swiftpm_app
fi

open_app() {
  /usr/bin/open -n "$APP_BUNDLE"
}

case "$MODE" in
  run)
    open_app
    ;;
  --debug|debug)
    lldb -- "$APP_BINARY"
    ;;
  --logs|logs)
    open_app
    /usr/bin/log stream --info --style compact --predicate "process == \"$APP_NAME\""
    ;;
  --telemetry|telemetry)
    open_app
    /usr/bin/log stream --info --style compact --predicate "subsystem == \"$BUNDLE_ID\""
    ;;
  --verify|verify)
    open_app
    sleep 1
    pgrep -x "$APP_NAME" >/dev/null
    ;;
  *)
    echo "usage: $0 [run|--debug|--logs|--telemetry|--verify]" >&2
    exit 2
    ;;
esac
