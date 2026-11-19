#!/bin/bash
# 给 go 命令接上「打过补丁的 MyGo」:把模块缓存里的 MyGo v0.10.0 复制一份到 target/mygo-patched,
# 用 mygo/_patches/ 里的文件覆盖同名源文件,再用 -modfile 把依赖指过去。模块缓存本身不动,仓库里只放补丁文件(几个文件,不是整个框架)。
# 用法:在其他脚本里 `source scripts/mygo-overlay.sh`(之后的 go build / vet / test 都带补丁);不 source 也能编译,只是没有补丁带来的行为。
# 补丁只改行为、不新增应用要调用的 API,所以未打补丁时依然能编译。
# 补丁清单(目标文件相对 MyGo 模块根):
#   ui/markdown/view.go ← mygo/_patches/ui-markdown-view.go   带 mc-path: 前缀的链接悬停不画下划线、改手型光标 + 淡底
# 升级 MyGo 版本后要对着新版本重做这些补丁(diff 旧版同名文件即可看到改了什么)。
_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
_dir="$(cd "$_root/mygo" && go list -m -f '{{.Dir}}' github.com/egoist/mygo 2>/dev/null)"
if [ -n "$_dir" ] && [ -d "$_dir" ]; then
  _out="$_root/target/mygo-patched"
  # 补丁改了 / MyGo 版本换了才重做副本
  _stamp="$_dir|$(cat "$_root"/mygo/_patches/* | shasum | cut -c1-16)"
  if [ "$(cat "$_out/.stamp" 2>/dev/null)" != "$_stamp" ]; then
    rm -rf "$_out"
    mkdir -p "$_root/target"
    cp -R "$_dir" "$_out"
    chmod -R u+w "$_out"
    cp "$_root/mygo/_patches/ui-markdown-view.go" "$_out/ui/markdown/view.go"
    echo "$_stamp" >"$_out/.stamp"
  fi
  cp "$_root/mygo/go.mod" "$_root/target/go.patched.mod"
  cp "$_root/mygo/go.sum" "$_root/target/go.patched.sum"
  printf '\nreplace github.com/egoist/mygo => %s\n' "$_out" >>"$_root/target/go.patched.mod"
  export GOFLAGS="${GOFLAGS:+$GOFLAGS }-modfile=$_root/target/go.patched.mod"
fi
unset _root _dir _out _stamp
