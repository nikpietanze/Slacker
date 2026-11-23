#!/bin/sh
# Builds "Slacker Companion.app" (Apple silicon + Intel) and zips it as
# dist/Slacker-Companion-macOS.zip. Run on macOS with Go and the Xcode
# command line tools: the menu bar icon needs cgo, so it can't be
# cross-compiled from Linux or Windows.
#
#   scripts/build-macos.sh [version]
set -eu

VERSION=${1:-dev}
ROOT=$(cd "$(dirname "$0")/.." && pwd)
DIST="$ROOT/dist"
APP="$DIST/Slacker Companion.app"
BIN="$APP/Contents/MacOS/slacker-companion"

rm -rf "$APP" "$DIST/Slacker-Companion-macOS.zip"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"

cd "$ROOT/companion"
for arch in arm64 amd64; do
	CGO_ENABLED=1 GOOS=darwin GOARCH=$arch go build -trimpath \
		-ldflags "-s -w -X main.version=$VERSION" -o "$DIST/slacker-companion-$arch" .
done
lipo -create -output "$BIN" "$DIST/slacker-companion-arm64" "$DIST/slacker-companion-amd64"
rm "$DIST/slacker-companion-arm64" "$DIST/slacker-companion-amd64"

iconutil -c icns -o "$APP/Contents/Resources/Slacker.icns" assets/Slacker.iconset

SHORT=$(echo "$VERSION" | sed 's/^v//')
cat > "$APP/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>CFBundleName</key><string>Slacker Companion</string>
  <key>CFBundleDisplayName</key><string>Slacker Companion</string>
  <key>CFBundleIdentifier</key><string>io.github.nikpietanze.slacker</string>
  <key>CFBundleExecutable</key><string>slacker-companion</string>
  <key>CFBundleIconFile</key><string>Slacker</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>$SHORT</string>
  <key>CFBundleVersion</key><string>$SHORT</string>
  <key>LSMinimumSystemVersion</key><string>11.0</string>
  <key>LSUIElement</key><true/>
  <key>NSHighResolutionCapable</key><true/>
</dict></plist>
EOF

# Ad-hoc signature so Apple silicon runs it. Without a Developer ID it still
# needs "Open Anyway" in System Settings > Privacy & Security the first time.
codesign --force --deep --sign - "$APP"

cd "$DIST"
ditto -c -k --keepParent "Slacker Companion.app" Slacker-Companion-macOS.zip
echo "wrote $DIST/Slacker-Companion-macOS.zip"
