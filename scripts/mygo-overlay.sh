#!/bin/bash
# 给 go 命令接上「打过补丁的 MyGo」:把模块缓存里的 MyGo 复制一份到 target/mygo-patched,
# 用 mygo/_patches/*.patch(统一 diff,相对上游文件)打上补丁,再用 -modfile 把依赖指过去。
# 模块缓存本身不动,仓库里只放几十行的 diff,不是整份上游文件。
#
# 用法:在其他脚本里 `source scripts/mygo-overlay.sh`(之后的 go build / vet / test 都带补丁);
# 不 source 也能编译,只是没有补丁带来的行为(补丁只改行为、不新增应用要调用的 API)。
#
# 补丁清单(路径相对 MyGo 模块根,每个补丁只改一个文件、互不重叠):
#   ui-markdown-view.patch   ui/markdown/view.go   带 mc-path: 前缀的链接悬停不画下划线、改手型光标 + 淡底
#   ui-markdown-spans.patch  ui/markdown/spans.go  带 mc-path: 前缀的链接:整枚路径胶囊由 LinkIcon 对象自画,链接文字不再另画
#   ui-markdown-table.patch  ui/markdown/table.go  表格比容器窄时撑满容器(多出的宽度按列宽比例分给各列)
#
# 基线校验:补丁用 patch -F0(零模糊度)应用,上下文对不上就失败。所以升级 MyGo 后只要补丁涉及的地方上游改过,
# 这里就会报错,而不会悄悄用旧代码覆盖上游。失败时:一份补丁都不打(补丁之间有依赖,半打会编不过),
# 退回上游原版并在 stderr 大声提示;设了 MINI_PATCH_STRICT=1(package.sh 这样用)则直接退出,不出没打补丁的包。
#
# 改补丁 / 升级 MyGo 后重做补丁:见 scripts/mygo-patch-export.sh。
_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
_dir="$(cd "$_root/mygo" && go list -m -f '{{.Dir}}' github.com/egoist/mygo 2>/dev/null)"
if [ -n "$_dir" ] && [ -d "$_dir" ]; then
  _out="$_root/target/mygo-patched"
  _bad="$_root/target/mygo-patched.failed" # 记下「这组 MyGo 版本 + 补丁」已知对不上,免得每次启动都重拷一遍
  _ver="$(basename "$_dir")"
  # 补丁改了 / MyGo 版本换了才重做副本
  _stamp="$_dir|$(cat "$_root"/mygo/_patches/*.patch | shasum | cut -c1-16)"
  mkdir -p "$_root/target"
  if [ "$(cat "$_out/.stamp" 2>/dev/null)" != "$_stamp" ] && [ "$(cat "$_bad" 2>/dev/null)" != "$_stamp" ]; then
    rm -rf "$_out" "$_bad"
    cp -R "$_dir" "$_out"
    chmod -R u+w "$_out"
    _failed=""
    for _p in "$_root"/mygo/_patches/*.patch; do
      if ! patch -p1 -F0 --no-backup-if-mismatch -s -d "$_out" <"$_p" >"$_root/target/mygo-patch.log" 2>&1; then
        _failed="$(basename "$_p")"
        break
      fi
    done
    if [ -z "$_failed" ]; then
      echo "$_stamp" >"$_out/.stamp"
    else
      rm -rf "$_out"
      echo "$_stamp" >"$_bad"
    fi
  fi
  if [ -f "$_out/.stamp" ]; then
    cp "$_root/mygo/go.mod" "$_root/target/go.patched.mod"
    cp "$_root/mygo/go.sum" "$_root/target/go.patched.sum"
    printf '\nreplace github.com/egoist/mygo => %s\n' "$_out" >>"$_root/target/go.patched.mod"
    export GOFLAGS="${GOFLAGS:+$GOFLAGS }-modfile=$_root/target/go.patched.mod"
  else
    {
      echo "⚠️  MyGo 补丁与 $_ver 对不上,本次没有打任何补丁(路径链接悬停样式、表格撑满宽度等行为会缺失)。"
      echo "    原因见 $_root/target/mygo-patch.log;补丁在 mygo/_patches/*.patch。"
      echo "    处理:对着新版本重做补丁(scripts/mygo-patch-export.sh 说明了怎么导出),或确认上游已内置后删掉对应补丁。"
    } >&2
    if [ -n "$MINI_PATCH_STRICT" ]; then
      echo "❌ MINI_PATCH_STRICT 已设置,拒绝在没有补丁的情况下继续。" >&2
      exit 1
    fi
  fi
fi
unset _root _dir _out _bad _ver _stamp _failed _p
