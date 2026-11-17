#!/bin/bash
# 重启开发版(MyGo 原生界面):先编译,成功后杀掉旧实例,再在最小 .app 包里启动唯一的新实例。
# 用法:scripts/dev.sh [环境变量…]   例:MINI_THEME=ink MINI_SCHEME=b scripts/dev.sh
# 日志:应用日志 ~/.minicode/logs/minicode.log;标准错误 / 帧统计 target/dev.log(上一个实例在 dev.log.prev)。
set -e
cd "$(dirname "$0")/.."
unset GOROOT
APP="$PWD/target/dev/MiniCode Dev.app"
PID_FILE="$PWD/target/dev.pid"
mkdir -p target/dev

# 先编译:失败时保留旧实例
(cd mygo && go vet ./... && go build -o "$PWD/../target/dev/minicode-new" .)

# 杀旧实例:pid 文件 + 路径兜底
[ -f "$PID_FILE" ] && kill "$(cat "$PID_FILE")" 2>/dev/null || true
pkill -if "MiniCode Dev.app/Contents/MacOS/minicode( |\$)" 2>/dev/null || true
sleep 0.5

# 组包(Info.plist 与图标清单和正式版共用 scripts/make-app.sh)
scripts/make-app.sh "$APP" target/dev/minicode-new "MiniCode Dev" dev.minicode.app.dev dev 0
rm -f target/dev/minicode-new
# 签名:先清掉扩展属性(resource fork / Finder 信息 / provenance 会让 codesign 报 detritus 失败),
# 否则包停在「只有链接器的临时签名、Identifier=a.out、Info.plist 未绑定」的状态,
# 系统通知授权会直接报 UNErrorDomain Code=1(通知不被允许),横幅也没有图标。失败要说出来,不再静默。
xattr -cr "$APP" 2>/dev/null || true
if ! codesign --force --sign - --identifier dev.minicode.app.dev "$APP" 2>target/codesign.err; then
  echo "警告:开发版签名失败(系统通知会不可用):$(tail -1 target/codesign.err)" >&2
fi
# 向 LaunchServices 登记这个包:系统通知的图标取自它登记的包内图标,没登记 / 换了包路径时横幅会显示成通用图标
/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister -f "$APP" >/dev/null 2>&1 || true

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
