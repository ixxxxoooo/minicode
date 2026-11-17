#!/usr/bin/env python3
"""生成锁定版式(图标 + 字标)PNG,给 README 与发布页用。

版式与设计稿 docs/design 的 U.brand.lockup 同一套算法:tile = 图标本体边长(图标本体 = 画布的 824/1024);
图标与字标间距 = tile / 8,字号 = tile × .58,字距 −2.5%;四周留 tile / 4 的透明安全边。
输出(docs/assets/):lockup-{light,dark}.png(横版)、lockup-v-{light,dark}.png(竖版)。
依赖:Google Chrome(headless 渲染文字,用系统 SF Pro)、Pillow。

用法:scripts/gen-lockup.py [tile 像素,默认 160]
@author ixxxxoooo"""
import os, subprocess, sys, tempfile
from PIL import Image

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
CHROME = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
TILE = int(sys.argv[1]) if len(sys.argv) > 1 else 160
BODY = 824 / 1024  # 图标本体占画布的比例

VARIANTS = {
    "light": ("app-style0.svg", "#1d1d1f"),
    "dark": ("app-style0-dark.svg", "#f2f2f4"),
}


def html(icon, ink, vertical):
    z = round(TILE / BODY)
    fs = TILE * 0.58
    direction = "column" if vertical else "row"
    return f"""<!doctype html><meta charset=utf-8><style>
html,body{{margin:0;background:transparent}}
.l{{display:inline-flex;flex-direction:{direction};align-items:center;gap:{TILE / 8}px;padding:{TILE}px;
font:600 {fs}px/1 -apple-system,'SF Pro Display',sans-serif;letter-spacing:-.025em;color:{ink};white-space:nowrap}}
.i{{width:{TILE}px;height:{TILE}px;position:relative}}
.i img{{position:absolute;left:50%;top:50%;width:{z}px;height:{z}px;translate:-50% -50%}}
.w{{padding-bottom:{fs * 0.06}px}}
</style><div class=l><div class=i><img src="file://{ROOT}/assets/icon/{icon}"></div><div class=w>MiniCode</div></div>"""


def render(name, icon, ink, vertical):
    with tempfile.TemporaryDirectory() as d:
        page, raw = os.path.join(d, "p.html"), os.path.join(d, "raw.png")
        open(page, "w", encoding="utf-8").write(html(icon, ink, vertical))
        subprocess.run([CHROME, "--headless=new", "--disable-gpu", "--hide-scrollbars", "--force-device-scale-factor=1",
                        "--default-background-color=00000000", "--window-size=1900,1500",
                        f"--screenshot={raw}", f"file://{page}"], check=True, capture_output=True)
        im = Image.open(raw).convert("RGBA")
    # 先按「图标本体 + 字标」裁到内容(忽略图标投影以外的透明),再补 tile/4 的安全边
    a = im.getchannel("A").point(lambda v: 255 if v > 24 else 0)
    box = a.getbbox()
    im = im.crop(box)
    pad = TILE // 4
    out = Image.new("RGBA", (im.width + 2 * pad, im.height + 2 * pad), (0, 0, 0, 0))
    out.alpha_composite(im, (pad, pad))
    path = os.path.join(ROOT, "docs", "assets", name)
    out.save(path, optimize=True)
    print(path, out.size)


os.makedirs(os.path.join(ROOT, "docs", "assets"), exist_ok=True)
for key, (icon, ink) in VARIANTS.items():
    render(f"lockup-{key}.png", icon, ink, False)
    render(f"lockup-v-{key}.png", icon, ink, True)


def styles_strip():
    """五种图标样式 × 浅 / 深两排,README 的「图标」一节用:docs/assets/icon-styles.png。"""
    size, gap, pad = 176, 28, 36
    n = 5
    W = pad * 2 + n * size + (n - 1) * gap
    H = pad * 2 + 2 * size + gap
    out = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    for row, suffix in enumerate(("", "-dark")):
        for i in range(n):
            im = Image.open(os.path.join(ROOT, "assets", "icon", f"app-style{i}{suffix}.png")).convert("RGBA")
            im = im.resize((size, size), Image.LANCZOS)
            out.alpha_composite(im, (pad + i * (size + gap), pad + row * (size + gap)))
    path = os.path.join(ROOT, "docs", "assets", "icon-styles.png")
    out.save(path, optimize=True)
    print(path, out.size)


styles_strip()
