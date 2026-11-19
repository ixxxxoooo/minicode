#!/bin/bash
# 批量出「Go 版 / 设计稿」两份截图,放在 /tmp/minicode-cmp/ 下并排对照。
# 用法:scripts/compare.sh <页面> <形态> <方案:主题> [<方案:主题>…]
# 例:scripts/compare.sh work agent a:paper b:mist c:ink
set -e
root="$(cd "$(dirname "$0")/.." && pwd)"
page=$1; mode=$2; shift 2
mkdir -p /tmp/minicode-cmp
unset GOROOT
source "$root/scripts/mygo-overlay.sh"
(cd "$root/mygo" && go build -o /tmp/minicode-cmp/shot ./cmd/shot)
for v in "$@"; do
  s=${v%%:*}; t=${v##*:}
  MINI_KEEP_STDERR=1 /tmp/minicode-cmp/shot $SHOTFLAGS -scheme "$s" -theme "$t" -mode "$mode" -screen "$page" -o "/tmp/minicode-cmp/go-$page-$mode-$s-$t.png" >/dev/null
  "$root/scripts/proto-shot.sh" "$page" "$s" "$t" "$mode" "/tmp/minicode-cmp/proto-$page-$mode-$s-$t.png" >/dev/null
  echo "/tmp/minicode-cmp/{go,proto}-$page-$mode-$s-$t.png"
done
