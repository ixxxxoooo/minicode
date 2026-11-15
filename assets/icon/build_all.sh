#!/bin/sh
# 全量图标:六种样式 × 四种外观 PNG/icns、托盘、文档图标。需要 python3、PyObjC、Pillow、iconutil。
set -eu
cd "$(dirname "$0")"
python3 gen.py all

render_app() {
  svg=$1
  base=$2
  python3 render.py "$svg" "${base}.png" 1024
  SET=$(mktemp -d)/icon.iconset
  mkdir -p "$SET"
  for s in 16 32 128 256 512; do
    python3 render.py "$svg" "$SET/icon_${s}x${s}.png" "$s"
    python3 render.py "$svg" "$SET/icon_${s}x${s}@2x.png" "$((s * 2))"
  done
  iconutil -c icns "$SET" -o "${base}.icns"
}

# 默认包内图标
render_app app-light.svg MiniCode
render_app app-dark.svg MiniCode-dark
render_app app-light.svg app-light
render_app app-dark.svg app-dark

for si in 0 1 2 3 4 5; do
  for suf in "" -dark -clear -tint; do
    svg="app-style${si}${suf}.svg"
    [ -f "$svg" ] || continue
    base="app-style${si}${suf}"
    render_app "$svg" "$base"
  done
done

for svg in tray-*.svg; do
  [ -f "$svg" ] || continue
  out="${svg%.svg}.png"
  tpl=""
  [ "$svg" != "tray.svg" ] && tpl="--template"
  python3 render.py "$svg" "$out" 36 $tpl
done

for doc in doc-mcworkspace doc-mccanvas doc-mcsettings; do
  [ -f "${doc}.svg" ] || continue
  render_app "${doc}.svg" "$doc"
done

echo "完成:$(ls -1 *.icns 2>/dev/null | wc -l) 个 icns,$(ls -1 tray-*.png 2>/dev/null | wc -l) 个托盘 PNG"
ls -lh MiniCode.icns app-style2.icns tray-error.png tray-running-0.png doc-mcworkspace.icns 2>/dev/null || true
# 同步到 Go 内嵌目录:托盘全部 PNG + 各样式 / 外观的 512px 应用图标(运行时不再依赖外部文件路径)
DEST=../../mygo/internal/app/brandassets
mkdir -p "$DEST"
rm -f "$DEST"/*.png
cp tray-*.png "$DEST"/
python3 - <<'PY'
import glob, os
from PIL import Image
dest = "../../mygo/internal/app/brandassets"
for p in sorted(glob.glob("app-style*.png")):
    Image.open(p).convert("RGBA").resize((512, 512), Image.LANCZOS).save(os.path.join(dest, os.path.basename(p)), optimize=True)
print("已同步内嵌素材:", len(os.listdir(dest)), "个")
PY
