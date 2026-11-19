// Package kit 是设计稿 docs/design/css/base.css 里基础组件的原生实现:按钮、图标按钮、胶囊、徽标、
// 分段控件、开关、列表行、标签条等。全部只读语义令牌(theme.Tokens),所以换主题 / 方案不用改组件。
//
// 每个组件都是「返回 ui.Element 的函数」,可以继续链式调整尺寸 / 间距;需要点击时用返回值的 Clicked()。
package kit

import (
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// T 是当前令牌。窗口只在主线程读它;换主题时由应用整体替换(见 Use)。
var T = theme.Resolve(theme.Palettes[0], theme.Schemes[0], "")

// Use 设置当前令牌。
func Use(t *theme.Tokens) { T = t }

// 动效时长(设计稿 tokens.css 的 --dur-* 与 --dur-loop)。
var (
	DurInstant = Scale(80 * time.Millisecond)
	DurFast    = Scale(120 * time.Millisecond)
	DurBase    = Scale(180 * time.Millisecond)
	DurSlow    = Scale(260 * time.Millisecond)
	DurStatus  = 160 * time.Millisecond // 状态行回合结束淡出(设计稿 resp-time)
	DurMorph   = Scale(440 * time.Millisecond)
	DurLoop    = 1600 * time.Millisecond // 呼吸 / 旋转等循环动画一档
)

// 呼吸 / 旋转之类的循环动效是否播放(设置里「动效」关闭或系统减弱动态时为 false)。
var motionOn = true

// pulseOn 控制运行中文字的呼吸(设置「流光样式」选「无」时关闭)。
var pulseOn = true

// SetMotion 打开 / 关闭动效:关闭时所有时长令牌归零(过渡立即完成)、呼吸不再播放。
func SetMotion(on bool) {
	motionOn = on
	if on {
		DurInstant, DurFast, DurBase, DurSlow, DurStatus, DurMorph = Scale(80*time.Millisecond), Scale(120*time.Millisecond), Scale(180*time.Millisecond), Scale(260*time.Millisecond), Scale(160*time.Millisecond), Scale(440*time.Millisecond)
		DurLoop = 1600 * time.Millisecond
	} else {
		DurInstant, DurFast, DurBase, DurSlow, DurStatus, DurMorph = 0, 0, 0, 0, 0, 0
		DurLoop = 0
	}
	Fade = ui.ElementTransition{Colors: true, Duration: DurFast}
	thumbSpring = ui.Animation{Duration: DurBase, Ease: EaseSpring}
}

// motionScale 是调试用的动效放慢倍数(环境变量 MINI_MOTION_SCALE,如 8 = 慢 8 倍),方便逐帧截图核对;默认 1。
var motionScale = func() float64 {
	if v, err := strconv.ParseFloat(os.Getenv("MINI_MOTION_SCALE"), 64); err == nil && v >= 1 && v <= 50 {
		return v
	}
	return 1
}()

// Scale 把一个设计稿时长按调试放慢倍数换算(正常运行恒等)。
func Scale(d time.Duration) time.Duration { return time.Duration(float64(d) * motionScale) }

// MotionOn 报告动效是否开着。
func MotionOn() bool { return motionOn }

// SetPulse 设置运行中文字的呼吸开关。
func SetPulse(on bool) { pulseOn = on }

// 缓动曲线(--ease-*):cubic-bezier 的四个控制点。
var (
	EaseOut    = ui.CubicBezier(.16, 1, .3, 1)
	EaseIn     = ui.CubicBezier(.5, 0, .9, .3)
	EaseIO     = ui.CubicBezier(.65, 0, .35, 1)
	EaseSpring = ui.CubicBezier(.34, 1.4, .64, 1)
	EaseMorph  = ui.CubicBezier(.3, 0, .1, 1)
)

// Fade 是悬停类的颜色过渡(120 ms)。
var Fade = ui.ElementTransition{Colors: true, Duration: DurFast}

// ───────── 图标 ─────────

var (
	iconMu    sync.Mutex
	iconCache = map[string]*ui.SVG{}
)

// svgOf 取得(并缓存)某个图标的 SVG。设计稿的图标是 24 格线性图标,描边 1.7。
// unknownIcons 被请求过却不在图标表里的名字(iconMu 保护)。
var unknownIcons = map[string]bool{}

// UnknownIcons 返回迄今请求过但图标表里没有的图标名(排序后),供测试检查。
func UnknownIcons() []string {
	iconMu.Lock()
	defer iconMu.Unlock()
	out := make([]string, 0, len(unknownIcons))
	for n := range unknownIcons {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func svgOf(name string) *ui.SVG {
	iconMu.Lock()
	defer iconMu.Unlock()
	if s, ok := iconCache[name]; ok {
		return s
	}
	body, ok := iconPaths[name]
	if !ok {
		body = iconPaths["x"] // 不存在的名字画成 ✕:记下来,单测会据此报出,不让它悄悄变成「图标显示异常」
		unknownIcons[name] = true
	}
	src := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`
	s := ui.MustParseSVG([]byte(src))
	iconCache[name] = s
	return s
}

// IconSVG 取某个图标的 SVG(给自绘用:Painter.Icon 画在胶囊等自绘元素里)。
func IconSVG(name string) *ui.SVG { return svgOf(name) }

// 图标尺寸:对应设计稿 svg.ic 的默认 14px 与 .s12 / .s16 / .s20。
const (
	S12 = 12
	S14 = 14
	S16 = 16
	S20 = 20
)

// Ic 画一个图标,颜色继承文字色(用 TextColor 改)。
func Ic(c *ui.Context, name string, size float32) ui.Element {
	size = theme.Px(size) // 图标随界面大小缩放
	return ui.Icon(c, svgOf(name)).Size(size, size)
}

// ───────── 文字 ─────────

// Txt 是带颜色与字号的单行文字。
func Txt(c *ui.Context, s string, size float32, col ui.Color) ui.Element {
	return ui.Text(c, s).FontSize(size).TextColor(col)
}

// Mono 是等宽文字(设计稿 .mono:11.5px)。
func Mono(c *ui.Context, s string, col ui.Color) ui.Element {
	return ui.Text(c, s).Font(theme.CodeFont()).FontSize(theme.Px(11.5)).TextColor(col)
}
