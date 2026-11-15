#!/bin/bash
# 重启开发版(MyGo 原生界面):先编译,成功后杀掉旧实例,再在最小 .app 包里启动唯一的新实例。
# 用法:scripts/dev.sh [环境变量…]   例:MINI_THEME=ink MINI_SCHEME=b scripts/dev.sh
# 日志:应用日志 ~/.minicode/logs/minicode.log;标准错误 / 帧统计 target/dev.log(上一个实例在 dev.log.prev)。
set -e
cd "$(dirname "$0")/.."
unset GOROOT
APP="$PWD/target/dev/MiniCode Dev.app"
BIN="$APP/Contents/MacOS/minicode"
PID_FILE="$PWD/target/dev.pid"
mkdir -p target/dev

# 先编译:失败时保留旧实例
(cd mygo && go vet ./... && go build -o "$PWD/../target/dev/minicode-new" .)

# 杀旧实例:pid 文件 + 路径兜底
[ -f "$PID_FILE" ] && kill "$(cat "$PID_FILE")" 2>/dev/null || true
pkill -if "MiniCode Dev.app/Contents/MacOS/minicode( |\$)" 2>/dev/null || true
sleep 0.5

mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
rm -f "$BIN"
mv target/dev/minicode-new "$BIN"
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
  <key>CFBundleName</key><string>MiniCode Dev</string>
  <key>CFBundleDisplayName</key><string>MiniCode Dev</string>
  <key>CFBundleIdentifier</key><string>dev.minicode.app.dev</string>
  <key>CFBundleExecutable</key><string>minicode</string>
  <key>CFBundleIconFile</key><string>MiniCode</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>dev</string>
  <key>CFBundleVersion</key><string>0</string>
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
codesign --force --sign - "$APP" >/dev/null 2>&1 || true

[ -f target/dev.log ] && mv -f target/dev.log target/dev.log.prev
: >target/dev.log
# 经 launchd 的 open 启动:进程不挂在当前终端会话下,脚本结束 / 终端关闭也不会连带被杀
# 用数组传 --env,值里带空格也完整(例:MINI_TERM_CMD='clear; ls -la')
ENVS=(--env MINI_DEV=1)
while IFS= read -r kv; do
  case "$kv" in MINI_DEV=*) ;; MINI_*=*) ENVS+=(--env "$kv") ;; esac
done < <(env)
open -n "$APP" --stdout "$PWD/target/dev.log" --stderr "$PWD/target/dev.log" "${ENVS[@]}" --args "$@"
sleep 1
pgrep -if "MiniCode Dev.app/Contents/MacOS/minicode( |\$)" | head -1 >"$PID_FILE"
echo "开发版已启动(pid $(cat "$PID_FILE"))"
