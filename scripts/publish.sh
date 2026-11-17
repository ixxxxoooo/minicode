#!/bin/bash
# 发布正式版:在干净的 HEAD 检出里打包 dmg → 生成 appcast.json → 打 tag → 推送 → 创建 GitHub Release。
# 应用内「检查更新」读取 releases/latest/download/appcast.json,发布后自动生效。
#
# 用法:先在 mygo/main.go 把 var version 改成新版本(如 "0.3.1-mygo")并提交推送,再运行  scripts/publish.sh
#   MINI_NOTES="更新说明"        可选,缺省取上一个 tag 以来的提交标题(最多 30 条)
#   DEVELOPER_ID="Developer ID Application: 名字 (TEAMID)"   可选,缺省为临时签名
#   MINI_DRY_RUN=1               只打包并生成 appcast,不打 tag、不推送、不创建 Release
# 打包用 git worktree 检出 HEAD,不受工作区里未提交改动影响(发布的就是已提交的代码)。
set -eu
cd "$(dirname "$0")/.."
ROOT="$PWD"

REPO="ixxxxoooo/minicode"
VERSION=$(sed -n 's/^var version = "\(.*\)"$/\1/p' mygo/main.go | head -1)
VERSION="${VERSION%-mygo}"
case "$VERSION" in [0-9]*.[0-9]*.[0-9]*) ;; *) echo "错误:版本号格式不正确:'$VERSION'" >&2; exit 1;; esac
TAG="v$VERSION"
git rev-parse "$TAG" >/dev/null 2>&1 && { echo "错误:tag $TAG 已存在,请先在 mygo/main.go 提升版本号并提交" >&2; exit 1; }
[ "${MINI_DRY_RUN:-}" = 1 ] || command -v gh >/dev/null || { echo "错误:需要 GitHub CLI(gh)并已登录" >&2; exit 1; }
git fetch -q origin
[ "$(git rev-parse HEAD)" = "$(git rev-parse '@{u}' 2>/dev/null || git rev-parse HEAD)" ] || { echo "错误:HEAD 与远端不一致,请先推送(发布的提交必须已在远端)" >&2; exit 1; }

BUILD=$(git rev-list --count HEAD)
PREV=$(git describe --tags --abbrev=0 2>/dev/null || true)
if [ -z "${MINI_NOTES:-}" ]; then
  MINI_NOTES=$(git log "${PREV:+$PREV..}HEAD" -n 30 --pretty='%s' | paste -sd ';' -)
fi

WT=$(mktemp -d "${TMPDIR:-/tmp}/mini-release.XXXXXX")
trap 'git -C "$ROOT" worktree remove --force "$WT" >/dev/null 2>&1 || true' EXIT
git worktree add -q --detach "$WT" HEAD
echo "==> 在干净检出里打包 $(git rev-parse --short HEAD)"
(cd "$WT" && MINI_BUILD="$BUILD" scripts/package.sh --dmg)
mkdir -p dist
DMG="dist/MiniCode-$VERSION.dmg"
rm -f dist/MiniCode*.dmg
cp "$WT/$DMG" "$DMG"
SHA=$(shasum -a 256 "$DMG" | awk '{print $1}')

# appcast:新版客户端读平铺字段,旧版(Rust)客户端读 stable 嵌套;两种都写。notes 用 python 转义。
VERSION="$VERSION" BUILD="$BUILD" SHA="$SHA" REPO="$REPO" TAG="$TAG" MINI_NOTES="$MINI_NOTES" python3 - <<'PY'
import json, os
e = os.environ
entry = {"version": e["VERSION"], "build": int(e["BUILD"]),
         "url": f'https://github.com/{e["REPO"]}/releases/download/{e["TAG"]}/MiniCode-{e["VERSION"]}.dmg',
         "sha256": e["SHA"], "notes": e["MINI_NOTES"]}
json.dump({**entry, "stable": entry}, open("dist/appcast.json", "w"), ensure_ascii=False, indent=2)
PY
echo "==> dmg: $DMG ($(du -sh "$DMG" | cut -f1)) sha256=$SHA"

if [ "${MINI_DRY_RUN:-}" = 1 ]; then
  echo "==> 演练完成(MINI_DRY_RUN=1):未打 tag、未推送、未创建 Release。产物在 dist/"
  exit 0
fi

git tag -a "$TAG" -m "MiniCode $VERSION"
git push origin "$TAG"
gh release create "$TAG" --repo "$REPO" --title "MiniCode $VERSION" \
  --notes "$(printf '%s\n' "$MINI_NOTES" | tr ';' '\n' | sed 's/^/- /')" --latest \
  "$DMG" dist/appcast.json
echo "已发布:https://github.com/$REPO/releases/tag/$TAG"
