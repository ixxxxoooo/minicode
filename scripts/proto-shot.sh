#!/bin/bash
# 截取设计稿原型的一个状态,用来与 Go 版截图一比一对照。
# 用法:scripts/proto-shot.sh <页面> <方案 a|b|c> <主题> <形态 agent|ide> <输出.png>
# 例:scripts/proto-shot.sh work a paper agent /tmp/proto-work-a-paper.png
set -e
page=${1:-work}; scheme=${2:-a}; theme=${3:-paper}; mode=${4:-agent}; out=${5:-/tmp/proto.png}
root="$(cd "$(dirname "$0")/.." && pwd)"
chrome="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
url="file://$root/docs/design/index.html#page=$page&scheme=$scheme&theme=$theme&mode=$mode&bare=1"
"$chrome" --headless=new --disable-gpu --hide-scrollbars --force-device-scale-factor=2 \
  --window-size=1280,800 --virtual-time-budget=${PROTO_BUDGET:-6000} --screenshot="$out" "$url" >/dev/null 2>&1
echo "$out"
