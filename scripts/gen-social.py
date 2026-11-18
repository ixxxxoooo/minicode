#!/usr/bin/env python3
"""生成 GitHub 社交预览图(1280×640)→ docs/assets/social-preview.png。

素材:docs/assets 里的横版锁定版式与 Agent 形态截图;背景渐变取自截图四角的暖桃 → 淡紫,风格与 README 一致。
中文字体用系统自带的苹方(PingFang SC),路径在不同 macOS 版本上可能不同,找不到时按 PINGFANG 环境变量指定。
用法:python3 -I scripts/gen-social.py      (需要 Pillow)
上传:仓库 Settings → General → Social preview → Edit → Upload an image(只能网页手动传)。

@author ixxxxoooo
"""
import glob
import os
import sys

from PIL import Image, ImageDraw, ImageFilter, ImageFont

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
ASSETS = os.path.join(ROOT, "docs", "assets")
OUT = os.path.join(ASSETS, "social-preview.png")
W, H, S = 1280, 640, 2  # 目标尺寸;按 2 倍绘制再缩小,边缘更平滑
INK = (29, 29, 31)


def find_pingfang():
    """找苹方字体文件;返回路径。"""
    cands = [os.environ.get("PINGFANG", "")]
    cands += glob.glob("/System/Library/AssetsV2/com_apple_MobileAsset_Font*/*/AssetData/PingFang.ttc")
    cands += ["/System/Library/Fonts/PingFang.ttc"]
    for p in cands:
        if p and os.path.exists(p):
            return p
    sys.exit("找不到苹方字体,请用环境变量 PINGFANG 指定 .ttc 路径")


def font(path, size, idx):
    """取苹方的某个字重(3 常规 / 7 中等 / 11 半粗),size 为 1 倍下的像素。"""
    return ImageFont.truetype(path, size * S, index=idx)


def gradient(w, h, c0, c1):
    """左上到右下的对角渐变。"""
    base = Image.new("RGB", (w, h), c0)
    top = Image.new("RGB", (w, h), c1)
    diag = Image.new("L", (w, h))
    px = diag.load()
    for y in range(h):
        for x in range(w):
            px[x, y] = int(255 * ((x / w) * 0.5 + (y / h) * 0.5))
    return Image.composite(top, base, diag)


def rounded(im, radius):
    """圆角裁切。"""
    m = Image.new("L", im.size, 0)
    ImageDraw.Draw(m).rounded_rectangle((0, 0, im.size[0] - 1, im.size[1] - 1), radius, fill=255)
    out = im.convert("RGBA")
    out.putalpha(m)
    return out


def main():
    pf = find_pingfang()
    cv = gradient(W * S, H * S, (255, 190, 150), (208, 181, 254)).convert("RGBA")

    # 右侧:Agent 形态截图,圆角 + 柔和阴影,右下出血
    shot = Image.open(os.path.join(ASSETS, "shot-agent.png")).convert("RGB")
    sw = 800 * S
    shot = shot.resize((sw, int(shot.height * sw / shot.width)), Image.LANCZOS)
    shot = rounded(shot, 22 * S)
    sx, sy = 500 * S, 78 * S
    shadow = Image.new("RGBA", cv.size, (0, 0, 0, 0))
    ImageDraw.Draw(shadow).rounded_rectangle(
        (sx, sy + 14 * S, sx + shot.width, sy + shot.height + 14 * S), 22 * S, fill=(40, 20, 80, 90))
    cv = Image.alpha_composite(cv, shadow.filter(ImageFilter.GaussianBlur(26 * S)))
    cv.alpha_composite(shot, (sx, sy))

    # 左侧:锁定版式 + 标语 + 三个要点
    lock = Image.open(os.path.join(ASSETS, "lockup-light.png")).convert("RGBA")
    lw = 360 * S
    lock = lock.resize((lw, int(lock.height * lw / lock.width)), Image.LANCZOS)
    cv.alpha_composite(lock, (56 * S, 70 * S))

    d = ImageDraw.Draw(cv)
    y = 70 * S + lock.height + 30 * S
    d.text((64 * S, y), "macOS 原生的", font=font(pf, 34, 11), fill=INK)
    d.text((64 * S, y + 46 * S), "Claude Code 工作台", font=font(pf, 34, 11), fill=INK)
    d.text((64 * S, y + 108 * S), "Agent 与 IDE 共用一个窗口", font=font(pf, 20, 7), fill=(29, 29, 31, 190))

    # 要点胶囊
    chips = ["Go 原生界面", "无 WebView", "直连 claude"]
    cx, cy = 64 * S, y + 160 * S
    cf = font(pf, 16, 7)
    for t in chips:
        tw = d.textlength(t, font=cf)
        pw, ph = int(tw + 28 * S), 34 * S
        pill = Image.new("RGBA", cv.size, (0, 0, 0, 0))
        ImageDraw.Draw(pill).rounded_rectangle((cx, cy, cx + pw, cy + ph), ph // 2, fill=(255, 255, 255, 170))
        cv = Image.alpha_composite(cv, pill)
        d = ImageDraw.Draw(cv)
        d.text((cx + 14 * S, cy + (ph - 16 * S) // 2 - 3 * S), t, font=cf, fill=INK)
        cx += pw + 10 * S

    out = cv.convert("RGB").resize((W, H), Image.LANCZOS)
    out.save(OUT, optimize=True)
    print("已生成", OUT, "%.0f KB" % (os.path.getsize(OUT) / 1024))


if __name__ == "__main__":
    main()
