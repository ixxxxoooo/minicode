#!/bin/bash
# 把 target/mygo-patched(打过补丁的 MyGo 副本)里相对上游改过的文件,导出成 mygo/_patches/*.patch。
# 文件 ui/markdown/view.go 对应补丁 ui-markdown-view.patch(路径里的 / 换成 -,去掉 .go)。
#
# 改补丁的流程:
#   1. source scripts/mygo-overlay.sh 或跑一次 scripts/dev.sh,让 target/mygo-patched 就绪(已带现有补丁)
#   2. 直接在 target/mygo-patched/ 里改 MyGo 源码(想验证就在 mygo/ 下带 GOFLAGS 编译试)
#   3. scripts/mygo-patch-export.sh   导出;git diff 看补丁变化后提交
#   (副本只在补丁或 MyGo 版本变化时才重建,所以第 2 步的改动在导出前不会丢;导出后补丁变了,下次会自动重建)
#
# 升级 MyGo 后补丁对不上(overlay 会报警)的重做流程:
#   1. 在 mygo/ 里把 go.mod 的 MyGo 升到新版并 go mod download
#   2. rm -rf target/mygo-patched && cp -R "$(cd mygo && go list -m -f '{{.Dir}}' github.com/egoist/mygo)" target/mygo-patched && chmod -R u+w target/mygo-patched
#   3. cd target/mygo-patched && patch -p1 < ../../mygo/_patches/xxx.patch   (逐个;失败的会留 .rej,对着手工合并)
#   4. 合并完 scripts/mygo-patch-export.sh;上游若已内置某个补丁的效果,删掉那个 .patch
set -euo pipefail
cd "$(dirname "$0")/.."
root="$PWD"
up="$(cd mygo && go list -m -f '{{.Dir}}' github.com/egoist/mygo)"
out="$root/target/mygo-patched"
[ -d "$out" ] || { echo "没有 $out:先 source scripts/mygo-overlay.sh 或跑 scripts/dev.sh" >&2; exit 1; }

n=0
# diff -rq 列出内容不同的文件;排除我们自己的标记与补丁残留
while IFS= read -r rel; do
  case "$rel" in .stamp | *.rej | *.orig) continue ;; esac
  name="$(echo "${rel%.go}" | tr '/' '-').patch"
  # --label 固定成 a/<相对路径> b/<相对路径>,补丁里不带本机绝对路径,patch -p1 / git apply 都能用
  diff -u --label "a/$rel" --label "b/$rel" "$up/$rel" "$out/$rel" >"mygo/_patches/$name" || true
  echo "导出 mygo/_patches/$name($(grep -c '^[+-][^+-]' "mygo/_patches/$name") 行改动)"
  n=$((n + 1))
done < <(diff -rq "$up" "$out" 2>/dev/null | sed -n -E 's#^Files .* and '"$out"'/(.*) differ$#\1#p')

[ "$n" -gt 0 ] || echo "副本和上游完全一致,没有补丁可导出。"
# 提醒:仓库里有补丁、但副本里该文件已经和上游一致(补丁变成空的)
for p in mygo/_patches/*.patch; do
  [ -s "$p" ] || echo "⚠️  $p 是空的(对应文件已和上游一致),确认后可以删除" >&2
done
