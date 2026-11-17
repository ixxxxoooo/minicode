#!/bin/bash
# 组装 .app 包(开发版 dev.sh 与正式版 package.sh 共用,避免 Info.plist / 图标清单两处各写一份):
#   scripts/make-app.sh <App 路径> <可执行文件> <显示名> <BundleID> <版本显示串> <构建号>
# 例:scripts/make-app.sh dist/MiniCode.app target/release/minicode MiniCode dev.minicode.app 0.3.0 812
set -e
cd "$(dirname "$0")/.."
APP="$1"; BIN_SRC="$2"; NAME="$3"; BUNDLE_ID="$4"; VERSION="$5"; BUILD="$6"
[ -n "$APP" ] && [ -f "$BIN_SRC" ] && [ -n "$NAME" ] && [ -n "$BUNDLE_ID" ] && [ -n "$VERSION" ] && [ -n "$BUILD" ] || {
  echo "用法:$0 <App 路径> <可执行文件> <显示名> <BundleID> <版本显示串> <构建号>" >&2
  exit 1
}

mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
rm -f "$APP/Contents/MacOS/minicode"
cp "$BIN_SRC" "$APP/Contents/MacOS/minicode"
chmod +x "$APP/Contents/MacOS/minicode"
cp assets/icon/MiniCode.icns "$APP/Contents/Resources/MiniCode.icns"
# 品牌图标(托盘各状态 PNG、深色 / 各样式 Dock 图标)放进包内 Resources,运行时按可执行文件位置查找
for f in assets/icon/*.png assets/icon/*.icns; do cp "$f" "$APP/Contents/Resources/"; done
for ic in doc-mcworkspace doc-mccanvas doc-mcsettings; do
  [ -f "assets/icon/${ic}.icns" ] && cp "assets/icon/${ic}.icns" "$APP/Contents/Resources/${ic}.icns"
done
cat > "$APP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>$NAME</string>
  <key>CFBundleDisplayName</key><string>$NAME</string>
  <key>CFBundleIdentifier</key><string>$BUNDLE_ID</string>
  <key>CFBundleExecutable</key><string>minicode</string>
  <key>CFBundleIconFile</key><string>MiniCode</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>$VERSION</string>
  <key>CFBundleVersion</key><string>$BUILD</string>
  <key>LSMinimumSystemVersion</key><string>13.0</string>
  <key>NSHighResolutionCapable</key><true/>
  <key>CFBundleDevelopmentRegion</key><string>zh_CN</string>
  <key>CFBundleDocumentTypes</key>
  <array>
    <dict>
      <key>CFBundleTypeName</key><string>MiniCode 工作区</string>
      <key>CFBundleTypeRole</key><string>Editor</string>
      <key>LSItemContentTypes</key>
      <array><string>io.minicode.workspace</string></array>
      <key>CFBundleTypeIconFile</key><string>doc-mcworkspace</string>
      <key>LSHandlerRank</key><string>Owner</string>
    </dict>
    <dict>
      <key>CFBundleTypeName</key><string>MiniCode 画布</string>
      <key>CFBundleTypeRole</key><string>Editor</string>
      <key>LSItemContentTypes</key>
      <array><string>io.minicode.canvas</string></array>
      <key>CFBundleTypeIconFile</key><string>doc-mccanvas</string>
      <key>LSHandlerRank</key><string>Owner</string>
    </dict>
    <dict>
      <key>CFBundleTypeName</key><string>MiniCode 设置备份</string>
      <key>CFBundleTypeRole</key><string>Viewer</string>
      <key>LSItemContentTypes</key>
      <array><string>io.minicode.settings-backup</string></array>
      <key>CFBundleTypeIconFile</key><string>doc-mcsettings</string>
      <key>LSHandlerRank</key><string>Owner</string>
    </dict>
  </array>
  <key>UTExportedTypeDeclarations</key>
  <array>
    <dict>
      <key>UTTypeIdentifier</key><string>io.minicode.workspace</string>
      <key>UTTypeDescription</key><string>MiniCode 工作区</string>
      <key>UTTypeConformsTo</key><array><string>public.data</string><string>public.content</string></array>
      <key>UTTypeTagSpecification</key>
      <dict><key>public.filename-extension</key><array><string>mcworkspace</string></array></dict>
    </dict>
    <dict>
      <key>UTTypeIdentifier</key><string>io.minicode.canvas</string>
      <key>UTTypeDescription</key><string>MiniCode 画布</string>
      <key>UTTypeConformsTo</key><array><string>public.data</string></array>
      <key>UTTypeTagSpecification</key>
      <dict><key>public.filename-extension</key><array><string>mccanvas</string></array></dict>
    </dict>
    <dict>
      <key>UTTypeIdentifier</key><string>io.minicode.settings-backup</string>
      <key>UTTypeDescription</key><string>MiniCode 设置备份</string>
      <key>UTTypeConformsTo</key><array><string>public.data</string></array>
      <key>UTTypeTagSpecification</key>
      <dict><key>public.filename-extension</key><array><string>mcsettings</string></array></dict>
    </dict>
  </array>
</dict>
</plist>
PLIST
