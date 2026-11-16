#!/bin/sh
# 主题配色版应用图标:四套主题配色(mist / celadon / graphite / aurora)× 六种样式 × 四种外观,
# 渲染成 512px PNG 直接放进 Go 内嵌素材目录(paper / ink 沿用默认的 app-style*.png)。
# SVG 只是中间产物,放在临时目录,不进仓库。需要 python3、PyObjC、Pillow。
set -eu
cd "$(dirname "$0")"
TMP=$(mktemp -d)
python3 gen.py all --theme-dir "$TMP/svg" >/dev/null
DEST=../../mygo/internal/app/brandassets
mkdir -p "$DEST"
for svg in "$TMP"/svg/app-*-style*.svg; do
  base=$(basename "$svg" .svg)
  python3 render.py "$svg" "$TMP/$base.png" 1024
done
python3 - "$TMP" "$DEST" <<'PY'
import glob, os, sys
from PIL import Image
tmp, dest = sys.argv[1], sys.argv[2]
n = 0
for p in sorted(glob.glob(os.path.join(tmp, "app-*-style*.png"))):
    Image.open(p).convert("RGBA").resize((512, 512), Image.LANCZOS).save(os.path.join(dest, os.path.basename(p)), optimize=True)
    n += 1
print("已同步主题配色应用图标:", n, "个")
PY
rm -rf "$TMP"
