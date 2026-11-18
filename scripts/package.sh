#!/bin/bash
# 打包正式版 macOS 应用(MyGo,通用二进制,部署目标 13.0):scripts/package.sh [--dmg]
# 产物 dist/MiniCode.app;加 --dmg 同时生成 dist/MiniCode-<版本>.dmg(应用内更新下载的就是它)。
# 版本号取 mygo/main.go 的 version(去掉 -mygo 后缀),编进二进制并写进 Info.plist。
#   DEVELOPER_ID="Developer ID Application: 名字 (TEAMID)"   可选,缺省为临时签名(用户首次打开需右键 → 打开)
#   MINI_BUILD=<构建号>                                        可选,缺省取提交数(单调递增)
# 注意:按当前工作区文件编译;发布请用 scripts/publish.sh(它在干净的 HEAD 检出里调用本脚本)。
set -eu
cd "$(dirname "$0")/.."
unset GOROOT

VERSION=$(sed -n 's/^var version = "\(.*\)"$/\1/p' mygo/main.go | head -1)
VERSION="${VERSION%-mygo}"
case "$VERSION" in [0-9]*.[0-9]*.[0-9]*) ;; *) echo "错误:取不到版本号(mygo/main.go 的 var version):'$VERSION'" >&2; exit 1;; esac
BUILD="${MINI_BUILD:-$(git rev-list --count HEAD 2>/dev/null || echo 0)}"

echo "==> 编译发布版 (v$VERSION, build $BUILD, $(uname -m))"
mkdir -p dist
# 部署目标固定 13.0(否则链接器取当前 SDK 版本,二进制 minos 会变成构建机系统,低版本 Mac 上图标显示禁止符号);
# arm64 + x86_64 各编一份再 lipo 合成通用二进制,Intel 与 Apple Silicon 通用。
export MACOSX_DEPLOYMENT_TARGET=13.0 CGO_ENABLED=1
export CGO_CFLAGS="-mmacosx-version-min=13.0" CGO_LDFLAGS="-mmacosx-version-min=13.0"
for ARCH in arm64 amd64; do
  CCARCH=$([ "$ARCH" = amd64 ] && echo x86_64 || echo arm64)
  (cd mygo && GOARCH=$ARCH CC="clang -arch $CCARCH" go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "$PWD/../dist/minicode-$ARCH" .)
done
lipo -create dist/minicode-arm64 dist/minicode-amd64 -output dist/minicode-bin
rm -f dist/minicode-arm64 dist/minicode-amd64
MINOS=$(otool -l dist/minicode-bin | grep -A5 LC_BUILD_VERSION | awk '/minos/{print $2; exit}')
[ "$MINOS" = "13.0" ] || { echo "错误:二进制 minos=$MINOS,应为 13.0" >&2; exit 1; }

APP=dist/MiniCode.app
rm -rf "$APP"
scripts/make-app.sh "$APP" dist/minicode-bin MiniCode dev.minicode.app "$VERSION" "$BUILD"
rm -f dist/minicode-bin

# 签名:设置 DEVELOPER_ID 用开发者证书(分发请再公证),否则临时签名
IDENTITY="${DEVELOPER_ID:--}"
if [ "$IDENTITY" = "-" ]; then
  codesign --force --deep --sign - "$APP" && echo "==> 已做临时签名"
else
  codesign --force --deep --options runtime --timestamp --sign "$IDENTITY" "$APP" && echo "==> 已用开发者证书签名"
fi
codesign --verify --deep "$APP" && echo "==> 签名校验通过"

if [ "${1:-}" = "--dmg" ]; then
  # 暂存目录放 app 与「Applications」快捷方式,用户可直接拖动安装;产物带版本号,供更新源使用
  STAGE=$(mktemp -d "${TMPDIR:-/tmp}/mini-dmg.XXXXXX")
  trap 'rm -rf "$STAGE"' EXIT
  ditto "$APP" "$STAGE/MiniCode.app"
  ln -s /Applications "$STAGE/Applications"
  DMG="dist/MiniCode-$VERSION.dmg"
  rm -f dist/MiniCode*.dmg
  hdiutil create -volname "MiniCode $VERSION" -srcfolder "$STAGE" -ov -format UDZO "$DMG" >/dev/null
  echo "==> 已生成 $DMG"
fi
echo "==> 完成:$APP ($(du -sh "$APP" | cut -f1))"
