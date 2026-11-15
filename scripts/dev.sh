#!/bin/sh
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
</dict>
</plist>
PLIST
codesign --force --sign - "$APP" >/dev/null 2>&1 || true

[ -f target/dev.log ] && mv -f target/dev.log target/dev.log.prev
: >target/dev.log
# 经 launchd 的 open 启动:进程不挂在当前终端会话下,脚本结束 / 终端关闭也不会连带被杀
ENVS="--env MINI_DEV=1"
for kv in $(env | grep '^MINI_' | grep -v '^MINI_DEV='); do ENVS="$ENVS --env $kv"; done
# shellcheck disable=SC2086
open -n "$APP" --stdout "$PWD/target/dev.log" --stderr "$PWD/target/dev.log" $ENVS --args "$@"
sleep 1
pgrep -if "MiniCode Dev.app/Contents/MacOS/minicode( |\$)" | head -1 >"$PID_FILE"
echo "开发版已启动(pid $(cat "$PID_FILE"))"
