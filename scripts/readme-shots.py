#!/usr/bin/env python3
"""给 README 出「整个窗口 + 柔和渐变背景 + 阴影」的演示模式截图。

做法:编译当前代码 → 组一个 target/shots/MiniCode Shots.app → 对每个场景用环境变量启动一个演示模式实例
(MINI_THEME / MINI_SCHEME / MINI_MODE / MINI_SCREEN / MINI_TAB …),等界面稳定后用 screencapture -l 截下真实窗口
(带透明圆角、不带系统阴影),再用 Pillow 合成到渐变背景上,输出 docs/assets/shot-*.png。
截的是真实运行中的窗口,不是 cmd/shot 的无窗口渲染。会先杀掉所有「MiniCode Dev」与本脚本启动的实例。

用法:scripts/readme-shots.py [场景名…]   不带参数 = 全部;场景见下面 SCENES。
依赖:macOS 屏幕录制权限(给运行本脚本的终端 / 编辑器)、Pillow、PyObjC(Quartz)。
@author ixxxxoooo"""
import os, shutil, subprocess, sys, time
import Quartz
from PIL import Image, ImageChops, ImageDraw, ImageFilter

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
APP = os.path.join(ROOT, "target", "shots", "MiniCode Shots.app")
OUT = os.path.join(ROOT, "docs", "assets")
BUNDLE_ID = "dev.minicode.app.shots"

# 场景:名称 → (环境变量, 窗口宽, 窗口高, 背景色板)
SCENES = {
    "agent": ({"MINI_THEME": "paper", "MINI_SCHEME": "a", "MINI_MODE": "agent", "MINI_SESSION": "fixbug", "MINI_OPEN": "review"}, 1280, 800, "peach"),
    "ide": ({"MINI_THEME": "paper", "MINI_SCHEME": "a", "MINI_MODE": "ide", "MINI_SESSION": "fixbug", "MINI_ED": "goto:src/router.rs:17"}, 1280, 800, "sky"),
    "dark": ({"MINI_THEME": "ink", "MINI_SCHEME": "a", "MINI_MODE": "agent", "MINI_SESSION": "fixbug", "MINI_OPEN": "review"}, 1280, 800, "night"),
    "icons": ({"MINI_THEME": "paper", "MINI_SCHEME": "a", "MINI_MODE": "agent", "MINI_SCREEN": "settings", "MINI_TAB": "brand"}, 1180, 840, "lilac"),
}

# 背景:底色渐变 + 几团模糊色块(柔和的「网格渐变」)
PALETTES = {
    "peach": ((255, 226, 205), (232, 213, 255), [((255, 170, 120), .25, .2), ((200, 170, 255), .8, .75), ((255, 215, 160), .75, .15)]),
    "sky": ((214, 232, 255), (236, 222, 255), [((130, 190, 255), .2, .25), ((190, 160, 255), .85, .8), ((160, 235, 225), .7, .1)]),
    "lilac": ((236, 224, 255), (255, 226, 214), [((180, 150, 255), .2, .3), ((255, 180, 150), .85, .7), ((150, 210, 255), .6, .05)]),
    "night": ((24, 26, 58), (58, 28, 70), [((70, 80, 200), .2, .25), ((190, 70, 150), .85, .8), ((40, 150, 190), .7, .1)]),
}


def sh(*a, **k):
    return subprocess.run(a, check=k.pop("check", True), capture_output=True, text=True, **k)


def build():
    os.makedirs(os.path.dirname(APP), exist_ok=True)
    env = dict(os.environ)
    env.pop("GOROOT", None)
    bin_path = os.path.join(ROOT, "target", "shots", "minicode-new")
    subprocess.run(["go", "build", "-o", bin_path, "."], cwd=os.path.join(ROOT, "mygo"), env=env, check=True)
    subprocess.run([os.path.join(ROOT, "scripts", "make-app.sh"), APP, bin_path, "MiniCode", BUNDLE_ID, "dev", "0"], cwd=ROOT, check=True)
    os.remove(bin_path)
    sh("codesign", "--force", "--sign", "-", APP, check=False)


def kill_all():
    sh("pkill", "-if", r"MiniCode (Dev|Shots)\.app/Contents/MacOS/minicode( |$)", check=False)
    time.sleep(1.0)


def find_window(pid):
    for w in Quartz.CGWindowListCopyWindowInfo(Quartz.kCGWindowListOptionAll, Quartz.kCGNullWindowID):
        b = w["kCGWindowBounds"]
        if w.get("kCGWindowOwnerPID") == pid and w.get("kCGWindowIsOnscreen") and b["Width"] > 600:
            return w
    return None


def capture(env_vars, w, h, raw):
    kill_all()
    # 用全新的数据目录:否则会带上本机开发版里改过的偏好(图标样式、主题……),README 里的截图就不是默认外观了
    home = "/tmp/minicode-shots-home"
    shutil.rmtree(home, ignore_errors=True)
    envs = ["--env", f"MINI_HOME={home}", "--env", "MINI_DEV=1", "--env", "MINI_DEMO=1", "--env", f"MINI_WIN_W={w}", "--env", f"MINI_WIN_H={h}",
            "--env", "MINI_WIN_X=160", "--env", "MINI_WIN_Y=80"]
    for k, v in env_vars.items():
        envs += ["--env", f"{k}={v}"]
    sh("open", "-n", APP, *envs, "--args", "--demo", "--new-window")  # --new-window:不读也不写上次的窗口位置 / 会话,尺寸以环境变量为准
    pid, win = None, None
    for _ in range(60):
        time.sleep(0.5)
        r = sh("pgrep", "-if", r"MiniCode Shots\.app/Contents/MacOS/minicode( |$)", check=False).stdout.split()
        if r:
            pid = int(r[0])
            win = find_window(pid)
            if win:
                break
    if not win:
        raise SystemExit("没有等到窗口")
    sh("open", "-b", BUNDLE_ID)  # 切到前台,避免被别的窗口盖住
    time.sleep(5.0)  # 等入场动画与演示数据稳定
    win = find_window(pid) or win
    sh("screencapture", "-x", "-o", f"-l{win['kCGWindowNumber']}", raw)
    kill_all()


def backdrop(size, pal):
    c1, c2, blobs = PALETTES[pal]
    W, H = size
    # 对角线性渐变(numpy:沿 (x + y) 方向插值,四角无空洞)
    import numpy as np
    yy, xx = np.mgrid[0:H, 0:W]
    t = ((xx / W) * .6 + (yy / H) * .4)[..., None]
    arr = np.array(c1)[None, None, :] * (1 - t) + np.array(c2)[None, None, :] * t
    bg = Image.fromarray(arr.astype("uint8"), "RGB")
    for col, fx, fy in blobs:
        blob = Image.new("L", size, 0)
        r = int(min(W, H) * .42)
        ImageDraw.Draw(blob).ellipse((int(W * fx) - r, int(H * fy) - r, int(W * fx) + r, int(H * fy) + r), fill=190)
        blob = blob.filter(ImageFilter.GaussianBlur(r * .55))
        bg = Image.composite(Image.new("RGB", size, col), bg, blob)
    return bg.convert("RGBA")


def compose(raw, out, pal):
    win = Image.open(raw).convert("RGBA")
    px = win.width // 1280 or 1
    padx, pady = 110 * px, 96 * px
    size = (win.width + 2 * padx, win.height + 2 * pady)
    bg = backdrop(size, pal)
    # 阴影:大而淡的一层 + 贴边紧凑的一层
    alpha = win.getchannel("A")
    for blur, off, op in ((46 * px, 34 * px, 95), (10 * px, 6 * px, 70)):
        sh_a = Image.new("L", size, 0)
        sh_a.paste(alpha.point(lambda v, o=op: v * o // 255), (padx, pady + off))
        sh_a = sh_a.filter(ImageFilter.GaussianBlur(blur))
        bg = Image.composite(Image.new("RGBA", size, (20, 14, 40, 255)), bg, sh_a)
    # 窗口外沿一圈极细的高光边,让深色窗口在深色背景上也有轮廓
    rim = ImageChops.subtract(alpha.filter(ImageFilter.MaxFilter(3)), alpha.filter(ImageFilter.MinFilter(3)))
    bg.alpha_composite(win, (padx, pady))
    edge = Image.new("RGBA", size, (255, 255, 255, 0))
    edge.paste(Image.new("RGBA", win.size, (255, 255, 255, 55)), (padx, pady), rim)
    bg.alpha_composite(edge)
    bg.convert("RGB").save(out, optimize=True)
    print(out, bg.size)


def main():
    names = sys.argv[1:] or list(SCENES)
    build()
    os.makedirs(OUT, exist_ok=True)
    for n in names:
        env_vars, w, h, pal = SCENES[n]
        raw = f"/tmp/readme-raw-{n}.png"
        capture(env_vars, w, h, raw)
        compose(raw, os.path.join(OUT, f"shot-{n}.png"), pal)


if __name__ == "__main__":
    main()
