#!/usr/bin/env python3
"""MiniCode 图标 SVG 生成:六种样式 × 四种外观、托盘状态、文档图标、Icon Composer 分层。

与 docs/design/js/brand.js 同源令牌;分段织纹不依赖背景色挖洞。
渲染 PNG / icns:render.py + build_all.sh(iconutil)。

@author ixxxxoooo
"""
from __future__ import annotations

import argparse
import pathlib
import textwrap

OUT = pathlib.Path(__file__).parent
LAYERS = OUT / "layers"

# Go appearance.appIcon 下标 → 样式 id(与设计稿 brand.js::STYLES、settings_brand.appStyles 顺序一致,默认下标 0 = 织纹)
STYLES = ("weave", "warp", "prompt", "grid", "lines", "braces")
# 菜单栏图标样式(设计稿 TRAYS = STYLES 去掉 warp);下标即 appearance.trayStyle
TRAY_STYLES = ("weave", "prompt", "grid", "lines", "braces")
APPEARANCES = ("light", "dark", "clear", "tint")
TRAY_STATES = ("idle", "running", "waiting", "unread", "error", "update", "paused", "demo")

SQ = (
    "M254.1 100.0H769.9C792.5 100.0 812.3 110.6 825.1 128.6C837.9 146.6 843.3 169.0 839.6 190.7"
    "C835.9 212.4 823.2 232.2 803.4 244.9C783.6 257.6 759.8 263.0 738.1 259.3"
    "C716.4 255.6 696.6 242.9 683.9 223.1C671.2 203.3 665.8 179.5 669.5 157.8"
    "C673.2 136.1 685.9 116.3 705.7 103.6C725.5 90.9 749.3 85.5 771.0 89.2"
    "C792.7 92.9 812.5 105.6 825.2 125.4L825.1 128.6"
)  # 简化圆角方;正式图标仍用 rx 方

N, BAND, GAP = 3, 112, 26
SPAN = N * BAND + (N - 1) * GAP
ORIGIN = 512 - SPAN / 2
TAIL = 66
W2 = dict(n=2, band=170, gap=34, tail=106, cut=20, ir=16, cx=512)
W3 = dict(n=3, band=112, gap=26, tail=82, cut=14, ir=12, cx=512)
BRACE = "M438 322C378 322 372 352 372 398V446C372 482 352 502 312 512C352 522 372 542 372 578V626C372 672 378 702 438 702"


def band_pos(i, cfg=W3):
    o = cfg["cx"] - (cfg["n"] * cfg["band"] + (cfg["n"] - 1) * cfg["gap"]) / 2
    return o + i * (cfg["band"] + cfg["gap"])


def rp(x, y, w, h, tl, tr, br, bl):
    k = min(1, h / (tl + bl or 1), h / (tr + br or 1), w / (tl + tr or 1), w / (bl + br or 1))
    tl, tr, br, bl = tl * k, tr * k, br * k, bl * k
    return (
        f"M{x + tl} {y}H{x + w - tr}A{tr} {tr} 0 0 1 {x + w} {y + tr}"
        f"V{y + h - br}A{br} {br} 0 0 1 {x + w - br} {y + h}H{x + bl}"
        f"A{bl} {bl} 0 0 1 {x} {y + h - bl}V{y + tl}A{tl} {tl} 0 0 1 {x + tl} {y}Z"
    )


def segs(lo, hi, skips):
    out, a = [], lo
    for s, e in sorted(skips):
        if s > a:
            out.append((a, s))
        a = max(a, e)
    if hi > a:
        out.append((a, hi))
    return out


def weave_parts(cfg):
    n, band, gap, tail, cut, ir, cx = (
        cfg["n"],
        cfg["band"],
        cfg["gap"],
        cfg["tail"],
        cfg["cut"],
        cfg["ir"],
        cfg.get("cx", 512),
    )
    span = n * band + (n - 1) * gap
    o = cx - span / 2
    lo, hi = o - tail, o + span + tail
    rad, irad = band / 2, min(ir, band / 2)
    pos = lambda j: o + j * (band + gap)
    weft, warp = [], []
    for j in range(n):
        sk = []
        for ii in range(n):
            if (ii + j) % 2 == 0:
                sk.append((pos(ii) - cut, pos(ii) + band + cut))
        weft.append(
            "".join(
                rp(a, pos(j), b - a, band, rad if a == lo else irad, rad if b == hi else irad, rad if b == hi else irad, rad if a == lo else irad)
                for a, b in segs(lo, hi, sk)
            )
        )
    for ii in range(n):
        sk = []
        for j in range(n):
            if (ii + j) % 2 == 1:
                sk.append((pos(j) - cut, pos(j) + band + cut))
        warp.append(
            "".join(
                rp(pos(ii), a, band, b - a, rad if a == lo else irad, rad if a == lo else irad, rad if b == hi else irad, rad if b == hi else irad)
                for a, b in segs(lo, hi, sk)
            )
        )
    return warp, weft


def mix(a, b, t):
    def h(s):
        return int(s[1:3], 16), int(s[3:5], 16), int(s[5:7], 16)

    A, B = h(a), h(b)
    return "#" + "".join(f"{round(A[i] + (B[i] - A[i]) * t):02x}" for i in range(3))


PAPER = dict(
    ac=("#FF8A3D", "#EE4A00"),
    acd=("#FF9A5C", "#FF5A0F"),
    L=("#FFFEFB", "#EAE7DE", "#26251E"),
    D=("#2C2B27", "#151411", "#EDEAE2"),
)


def resolve_appearance(mode: str, dark: bool = False):
    p = PAPER
    th = p["acd"] if dark else p["ac"]
    if mode == "clear":
        return dict(
            bg0="#ffffff",
            bg1="#ffffff",
            bgo0=0.4,
            bgo1=0.12,
            ink="#ffffff",
            th0=mix(th[0], "#ffffff", 0.55),
            th1=mix(th[1], "#ffffff", 0.3),
            rim=0.85,
            shade=0.14,
            gloss=0,
        )
    if mode == "tint":
        t1 = p["acd"][1]
        return dict(
            bg0=mix(t1, "#000000", 0.38),
            bg1=mix(t1, "#000000", 0.74),
            bgo0=1,
            bgo1=1,
            ink=mix(p["acd"][0], "#ffffff", 0.84),
            th0=mix(p["acd"][0], "#ffffff", 0.2),
            th1=p["acd"][0],
            rim=0.22,
            shade=0.5,
            gloss=0.1,
        )
    n = p["D"] if dark else p["L"]
    return dict(
        bg0=n[0],
        bg1=n[1],
        bgo0=1,
        bgo1=1,
        ink=n[2],
        th0=th[0],
        th1=th[1],
        rim=0.16 if dark else 0.95,
        shade=0.5 if dark else 0.2,
        gloss=0.1 if dark else 0.5,
    )


def art_svg(style: str, ink: str, th_ref: str) -> str:
    if style == "weave":
        w, f = weave_parts(W3)
        return f'<path d="{"".join(w)}" fill="{ink}"/><path d="{"".join(f)}" fill="{th_ref}"/>'
    if style == "warp":
        w, f = weave_parts(W2)
        return f'<path d="{"".join(w)}" fill="{ink}"/><path d="{"".join(f)}" fill="{th_ref}"/>'
    if style == "prompt":
        return (
            f'<path d="M372 368 520 512 372 656" fill="none" stroke="{th_ref}" stroke-width="112"'
            f' stroke-linecap="round" stroke-linejoin="round"/>'
            f'<rect x="566" y="600" width="150" height="112" rx="56" fill="{ink}"/>'
        )
    if style == "grid":
        c, g, o = 118, 30, 512 - (3 * 118 + 2 * 30) / 2
        a, b = "", ""
        for i in range(3):
            for j in range(3):
                r = f'<rect x="{o + i * (c + g)}" y="{o + j * (c + g)}" width="{c}" height="{c}" rx="34"/>'
                (a if (i + j) % 2 == 0 else b).__class__  # noqa
                if (i + j) % 2 == 0:
                    a += r
                else:
                    b += r
        return f'<g fill="{th_ref}">{a}</g><g fill="{ink}">{b}</g>'
    if style == "lines":
        x0, h, gap, y0 = 300, 76, 44, 512 - (4 * 76 + 3 * 44) / 2
        rows = [[[0, 150, 1], [174, 250, 0]], [[96, 328, 0]], [[96, 190, 1], [214, 114, 0]], [[0, 230, 0], [254, 96, 1]]]

        def pill(x, y, w, hh):
            return f'<rect x="{x}" y="{y}" width="{w}" height="{hh}" rx="{hh / 2}"/>'

        a, b = "", ""
        for k, r in enumerate(rows):
            for x, w, t in r:
                s = pill(x0 + x, y0 + k * (h + gap), w, h)
                (a if t else b).__class__
                if t:
                    a += s
                else:
                    b += s
        return f'<g fill="{th_ref}">{a}</g><g fill="{ink}">{b}</g>'
    if style == "braces":
        return (
            f'<g fill="none" stroke="{ink}" stroke-width="96" stroke-linecap="round" stroke-linejoin="round">'
            f'<path d="{BRACE}"/><path d="{BRACE}" transform="translate(1024 0) scale(-1 1)"/></g>'
            f'<circle cx="512" cy="512" r="56" fill="{th_ref}"/>'
        )
    raise ValueError(style)


def app_icon_svg(style: str, mode: str, uid: str = "a") -> str:
    dark = mode == "dark" or (mode == "tint")
    r = resolve_appearance(mode, dark=mode in ("dark", "tint"))
    return f'''<svg xmlns="http://www.w3.org/2000/svg" width="1024" height="1024" viewBox="0 0 1024 1024">
<defs>
  <linearGradient id="{uid}b" x1="0" y1="100" x2="0" y2="924" gradientUnits="userSpaceOnUse">
    <stop offset="0" stop-color="{r["bg0"]}" stop-opacity="{r["bgo0"]}"/>
    <stop offset="1" stop-color="{r["bg1"]}" stop-opacity="{r["bgo1"]}"/></linearGradient>
  <linearGradient id="{uid}t" x1="250" y1="250" x2="774" y2="774" gradientUnits="userSpaceOnUse">
    <stop offset="0" stop-color="{r["th0"]}"/><stop offset="1" stop-color="{r["th1"]}"/></linearGradient>
  <linearGradient id="{uid}r" x1="0" y1="0" x2="0" y2="1">
    <stop offset="0" stop-color="#fff" stop-opacity="{r["rim"]}"/><stop offset="0.5" stop-color="#fff" stop-opacity="0"/></linearGradient>
  <filter id="{uid}sh" x="-20%" y="-20%" width="140%" height="140%">
    <feDropShadow dx="0" dy="12" stdDeviation="16" flood-color="#000" flood-opacity="{r["shade"]}"/></filter>
</defs>
<rect x="100" y="100" width="824" height="824" rx="186" fill="url(#{uid}b)" filter="url(#{uid}sh)"/>
<rect x="102" y="102" width="820" height="820" rx="184" fill="none" stroke="url(#{uid}r)" stroke-width="4"/>
  {art_svg(style, r["ink"], f"url(#{uid}t)")}
</svg>'''


def _lv(i: int, frame: int | None) -> str:
    """动画层:frame 为 None 时常亮;否则只有第 frame%3 层亮,其余 .26(与设计稿 brand.js 的 lv 同义)。"""
    op = 1.0 if frame is None or i == frame % 3 else 0.26
    return f'opacity="{op}"'


def tray_weave_body(frame: int | None = None) -> str:
    b, g, cut, tail, pad, n = 6, 3, 1, 3, 3, 3
    pos = [pad + tail + i * (b + g) for i in range(n)]
    lo, hi = pad, 36 - pad
    parts = []
    for j, y in enumerate(pos):
        parts.append(f'<rect x="{lo}" y="{y}" width="{hi - lo}" height="{b}" rx="{b / 2}" fill="#000" {_lv(j, frame)}/>')
    for x in pos:
        parts.append(f'<rect x="{x}" y="{lo}" width="{b}" height="{hi - lo}" rx="{b / 2}" fill="#000"/>')
    for i, x in enumerate(pos):
        for j, y in enumerate(pos):
            if (i + j) % 2 == 1:
                parts += [
                    f'<rect x="{x}" y="{y - cut}" width="{b}" height="{cut}" fill="#fff"/>',
                    f'<rect x="{x}" y="{y + b}" width="{b}" height="{cut}" fill="#fff"/>',
                ]
            else:
                parts += [
                    f'<rect x="{x - cut}" y="{y}" width="{cut}" height="{b}" fill="#fff"/>',
                    f'<rect x="{x + b}" y="{y}" width="{cut}" height="{b}" fill="#fff"/>',
                ]
    return "\n  ".join(parts)


def tray_body(style: str, frame: int | None = None) -> str:
    """五种菜单栏图标的主体(36 画布、单色黑;白底在渲染成模板图时转成透明)。移植自设计稿 brand.js::TRAY。"""
    if style == "weave":
        return tray_weave_body(frame)
    if style == "prompt":
        return (
            '<path d="M10.5 9.5 18 18l-7.5 8.5" fill="none" stroke="#000" stroke-width="4.4" '
            'stroke-linecap="round" stroke-linejoin="round"/>'
            f'<rect {_lv(1, frame)} x="20.5" y="23.6" width="8" height="4.4" rx="2.2" fill="#000"/>'
        )
    if style == "grid":
        c, g = 7.4, 2.4
        o = 18 - (3 * c + 2 * g) / 2
        out, k = "", 0
        for i in range(3):
            for j in range(3):
                geo = f'x="{o + i * (c + g):.2f}" y="{o + j * (c + g):.2f}" width="{c}" height="{c}" rx="2.2"'
                if (i + j) % 2 == 0:
                    out += f'<rect {geo} fill="#000"/>'
                else:
                    out += f'<rect {_lv(k, frame)} {geo} fill="#000" fill-opacity=".42"/>'
                    k += 1
        return out
    if style == "lines":
        sx, x0, h, gap = 27 / 424, 4.5, 4.4, 3.6
        y0 = 18 - (4 * h + 3 * gap) / 2
        rows = [[(0, 150, 1), (174, 250, 0)], [(96, 328, 0)], [(96, 190, 1), (214, 114, 0)], [(0, 230, 0), (254, 96, 1)]]
        out, k = "", 0
        for j, row in enumerate(rows):
            for x, w, t in row:
                geo = f'x="{x0 + x * sx:.2f}" y="{y0 + j * (h + gap):.2f}" width="{w * sx:.2f}" height="{h}" rx="{h / 2}"'
                if t:
                    out += f'<rect {_lv(k, frame)} {geo} fill="#000"/>'
                    k += 1
                else:
                    out += f'<rect {geo} fill="#000" fill-opacity=".55"/>'
        return out
    if style == "braces":
        return (
            '<g transform="translate(18 18) scale(.047) translate(-512 -512)">'
            '<g fill="none" stroke="#000" stroke-width="84" stroke-linecap="round" stroke-linejoin="round">'
            f'<path d="{BRACE}"/><path d="{BRACE}" transform="translate(1024 0) scale(-1 1)"/></g>'
            f'<circle {_lv(1, frame)} cx="512" cy="512" r="64" fill="#000"/></g>'
        )
    raise ValueError(style)


def tray_badge(state: str, style: str = "weave", frame: int | None = None) -> str:
    """带徽标的状态:主体先用圆形遮罩挖出一圈空白,再画徽标(demo 是空心圆环,paused 整体半透明)。"""
    x, y = 28.2, 8
    spec = {
        "waiting": (6.4, "!"),
        "unread": (4.6, None),
        "error": (6.4, "x"),
        "update": (6.4, "arr"),
        "paused": (6.4, "||"),
        "demo": (4.2, "ring"),
    }
    if state not in spec:
        return ""
    r, sym = spec[state]
    mask = f'<mask id="mk" maskUnits="userSpaceOnUse" x="0" y="0" width="36" height="36"><rect width="36" height="36" fill="#fff"/><circle cx="{x}" cy="{y}" r="{r + 2.3}" fill="#000"/></mask>'
    body_op = ' opacity=".5"' if state == "paused" else ""
    body = f'<g mask="url(#mk)"{body_op}>{tray_body(style, frame)}</g>'
    if sym == "ring":
        g = f'<circle cx="{x}" cy="{y}" r="3.7" fill="none" stroke="#000" stroke-width="1.9"/>'
    elif sym == "!":
        g = f'<circle cx="{x}" cy="{y}" r="{r}" fill="#000"/><rect x="{x - .95}" y="{y - 3.5}" width="1.9" height="4.2" rx=".95" fill="#fff"/><circle cx="{x}" cy="{y + 2.6}" r="1.1" fill="#fff"/>'
    elif sym == "x":
        g = f'<circle cx="{x}" cy="{y}" r="{r}" fill="#000"/><path d="M{x - 2.1} {y - 2.1}l4.2 4.2M{x + 2.1} {y - 2.1}l-4.2 4.2" stroke="#fff" stroke-width="1.7" stroke-linecap="round"/>'
    elif sym == "arr":
        g = f'<circle cx="{x}" cy="{y}" r="{r}" fill="#000"/><path d="M{x} {y - 3}v5.2M{x - 2.4} {y - .3}l2.4 2.6 2.4-2.6" stroke="#fff" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" fill="none"/>'
    elif sym == "||":
        g = f'<circle cx="{x}" cy="{y}" r="{r}" fill="#000"/><rect x="{x - 2.3}" y="{y - 2.6}" width="1.7" height="5.2" rx=".6" fill="#fff"/><rect x="{x + .6}" y="{y - 2.6}" width="1.7" height="5.2" rx=".6" fill="#fff"/>'
    else:
        g = f'<circle cx="{x}" cy="{y}" r="{r}" fill="#000"/>'
    return mask + body + g


def tray_icon_svg(state: str = "idle", frame: int | None = None, style: str = "weave") -> str:
    head = '<svg xmlns="http://www.w3.org/2000/svg" width="36" height="36" viewBox="0 0 36 36"><rect width="36" height="36" fill="#fff"/>'
    if state in ("waiting", "unread", "error", "update", "paused", "demo"):
        return head + "  " + tray_badge(state, style, frame) + "</svg>"
    return head + "  " + tray_body(style, frame if state == "running" else None) + "</svg>"


def tray_file(style_idx: int, state: str, frame: int | None = None) -> str:
    """托盘文件名:样式 0(织纹)沿用旧名 tray-<状态>,其余是 tray-s<下标>-<状态>。"""
    pre = "tray" if style_idx == 0 else f"tray-s{style_idx}"
    if frame is not None:
        return f"{pre}-{state}-{frame}.svg"
    return f"{pre}-{state}.svg"


def doc_icon_svg(label: str, style: str = "weave") -> str:
    r = resolve_appearance("light")
    uid = "d"
    art = art_svg(style, r["ink"], f"url(#{uid}t)")
    return f'''<svg xmlns="http://www.w3.org/2000/svg" width="1024" height="1024" viewBox="0 0 1024 1024">
<defs>
  <linearGradient id="{uid}p" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#ffffff"/><stop offset="1" stop-color="#eeece6"/></linearGradient>
  <linearGradient id="{uid}t" x1="250" y1="250" x2="774" y2="774" gradientUnits="userSpaceOnUse">
    <stop offset="0" stop-color="{r["th0"]}"/><stop offset="1" stop-color="{r["th1"]}"/></linearGradient>
</defs>
<path d="M252 80H664L812 228V904Q812 944 772 944H252Q212 944 212 904V120Q212 80 252 80Z" fill="url(#{uid}p)" stroke="rgba(0,0,0,.14)" stroke-width="3"/>
<path d="M664 80V188Q664 228 704 228H812Z" fill="#dedbd2" stroke="rgba(0,0,0,.14)" stroke-width="3"/>
<g transform="translate(512 500) scale(.5) translate(-512 -512)">{art}</g>
<rect x="282" y="770" width="460" height="104" rx="52" fill="url(#{uid}t)"/>
<text x="512" y="842" text-anchor="middle" font-family="-apple-system,SF Pro Text,sans-serif" font-weight="700" font-size="62" fill="#fff">{label}</text>
</svg>'''


def write_layers():
    LAYERS.mkdir(exist_ok=True)
    r = resolve_appearance("light")
    bg = f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024">
<defs><linearGradient id="bg" x1="0" y1="100" x2="0" y2="924" gradientUnits="userSpaceOnUse">
<stop offset="0" stop-color="{r["bg0"]}"/><stop offset="1" stop-color="{r["bg1"]}"/></linearGradient></defs>
<rect x="100" y="100" width="824" height="824" rx="186" fill="url(#bg)"/></svg>'''
    w, f = weave_parts(W3)
    warp = f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024"><path d="{"".join(w)}" fill="{r["ink"]}"/></svg>'
    weft = f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024"><defs>
<linearGradient id="th" x1="250" y1="250" x2="774" y2="774" gradientUnits="userSpaceOnUse">
<stop offset="0" stop-color="{r["th0"]}"/><stop offset="1" stop-color="{r["th1"]}"/></linearGradient></defs>
<path d="{"".join(f)}" fill="url(#th)"/></svg>'''
    (LAYERS / "background.svg").write_text(bg)
    (LAYERS / "warp.svg").write_text(warp)
    (LAYERS / "weft.svg").write_text(weft)
    note = textwrap.dedent(
        """\
        # MiniCode.icon(Icon Composer)
        三层 SVG 在 layers/:background / warp / weft。
        可用 Apple Icon Composer(macOS 26)导入后导出 .icon;本仓库未打包 .icon 目录(避免未签名包结构)。
        运行时 Dock 仍用 .icns / PNG;透明·着色外观用 app-style*-clear / -tint PNG。
        """
    )
    (LAYERS / "README.txt").write_text(note)


def generate_all():
    # 默认织纹浅 / 深(兼容旧名)
    (OUT / "app-light.svg").write_text(app_icon_svg("weave", "light", "l"))
    (OUT / "app-dark.svg").write_text(app_icon_svg("weave", "dark", "d"))
    for si, st in enumerate(STYLES):
        for mode in APPEARANCES:
            suf = "" if mode == "light" else f"-{mode}"
            if mode == "dark":
                suf = "-dark"
            name = f"app-style{si}{suf}.svg"
            (OUT / name).write_text(app_icon_svg(st, mode, f"s{si}{mode}"))
    for si, tst in enumerate(TRAY_STYLES):
        for state in TRAY_STATES:
            (OUT / tray_file(si, state)).write_text(tray_icon_svg(state, None, tst))
        for fr in range(3):
            (OUT / tray_file(si, "running", fr)).write_text(tray_icon_svg("running", fr, tst))
    (OUT / "tray.svg").write_text(tray_icon_svg("idle"))
    for lab, fn in (("MCWS", "doc-mcworkspace.svg"), ("CANVAS", "doc-mccanvas.svg"), ("CONF", "doc-mcsettings.svg")):
        (OUT / fn).write_text(doc_icon_svg(lab))
    write_layers()


def main():
    p = argparse.ArgumentParser(description="生成 MiniCode 图标 SVG")
    p.add_argument("cmd", nargs="?", default="all", choices=("all", "legacy", "tray", "layers"))
    p.add_argument("--style", type=int, default=2)
    p.add_argument("--mode", default="light")
    args = p.parse_args()
    if args.cmd == "all":
        generate_all()
    elif args.cmd == "legacy":
        (OUT / "app-light.svg").write_text(app_icon_svg("weave", "light"))
        (OUT / "app-dark.svg").write_text(app_icon_svg("weave", "dark"))
        (OUT / "tray.svg").write_text(tray_icon_svg("idle"))
    elif args.cmd == "tray":
        for si, tst in enumerate(TRAY_STYLES):
            for state in TRAY_STATES:
                (OUT / tray_file(si, state)).write_text(tray_icon_svg(state, None, tst))
    elif args.cmd == "layers":
        write_layers()


if __name__ == "__main__":
    main()