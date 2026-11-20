package theme

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
)

// 表面色调(设置 → 主题与布局 → 背景与表面):给主题里那几层中性灰(窗口底 / 面板 / 卡片 / 凹槽 / 墨色)一个色相。
// 对应设计稿 css/tint.css:明度梯度固定(各层之间的明暗差不变),只让灰带一点色相,所以文字对比度与原主题一致;
// 不动成功 / 报错 / 警告 / 语法高亮等语义色,也不动阴影基色。流光(玻璃)的染色 / 高光 / 玻璃底色同样随这个色相走。
//
// @author ixxxxoooo

// TintPreset 是一个表面色调预设。Hue 是 OKLCH 色相(度),K 是浓淡 0–100;Accent / AccentDark 是推荐强调色(浅 / 深)。
type TintPreset struct {
	Name               string
	Hue, K             int
	Accent, AccentDark string
}

// TintCustom 是「自定义」在预设表里的下标;下标 0 = 跟随主题(不套色调)。
const TintCustom = 8

// TintPresets 与设计稿 settings.js 的 W.TINTS 一一对应(参考 Radix 的 sand / sage / slate / mauve 带色相灰阶、Rosé Pine Dawn、Kanagawa、GitHub 中性灰)。
var TintPresets = []TintPreset{
	{Name: "跟随主题"},
	{"燕麦", 85, 40, "#BD5A36", "#E8967A"},
	{"鼠尾草", 150, 38, "#2F7D5B", "#6FCB9B"},
	{"石板", 255, 38, "#3D5FD0", "#7F9CFF"},
	{"藕荷", 320, 30, "#8A56C2", "#B994E8"},
	{"黛青", 215, 42, "#0E7C86", "#4CC3CF"},
	{"玫瑰灰", 10, 36, "#C0486A", "#EE8AA6"},
	{"石墨灰", 0, 0, "#4662E8", "#8AA0FF"},
	{Name: "自定义"},
}

// TintAccent 返回预设的推荐强调色(没有推荐色 = 空串:跟随主题与自定义)。
func TintAccent(idx int, dark bool) string {
	if idx <= 0 || idx >= len(TintPresets) || idx == TintCustom {
		return ""
	}
	if dark {
		return TintPresets[idx].AccentDark
	}
	return TintPresets[idx].Accent
}

// glassOv 是流光主题里随色相走的几个半透明色(只有 WithTint 会设置,见 Tone)。
type glassOv struct {
	// 浅色:壁纸之上的暖色染色与面板深度
	sideTint, wsTint, depth, chatTint ui.Color
	// 夜色:玻璃底色与高光
	thin, card, sel, rim, rimLo, sheen, mid, hi ui.Color
}

// WithTint 返回套了表面色调的主题副本:重写 Base / Surface / Raised / Sunken / Ink,玻璃主题另带染色覆盖。
// hue 为 OKLCH 色相(度),k 为浓淡 0–100(0 = 纯中性灰)。
func WithTint(p Palette, hue, k int) Palette {
	kk := float64(min(max(k, 0), 100)) / 100
	h := float64(hue)
	if p.Dark {
		p.Base, p.Surface, p.Raised, p.Sunken = oklchHex(.170, kk*.021, h), oklchHex(.205, kk*.018, h), oklchHex(.245, kk*.016, h), oklchHex(.185, kk*.020, h)
		p.Ink = oklchHex(.93, kk*.010, h)
	} else {
		p.Base, p.Surface, p.Raised, p.Sunken = oklchHex(.945, kk*.026, h), oklchHex(.978, kk*.014, h), oklchHex(1, kk*.004, h), oklchHex(.962, kk*.020, h)
		p.Ink = oklchHex(.22, kk*.012, h)
	}
	if p.Glass {
		a := func(l, c, al float64) ui.Color {
			r := oklchRGB(l, kk*c, h)
			return ui.RGBA(r[0], r[1], r[2], float32(al))
		}
		if p.Dark {
			p.glass = &glassOv{thin: a(.94, .025, .08), card: a(.33, .014, .46), sel: a(.94, .025, .16), rim: a(.95, .015, .30), rimLo: a(.95, .015, .09),
				sheen: a(.95, .020, .14), mid: a(.27, .012, .60), hi: a(.96, .015, .16)}
		} else {
			p.glass = &glassOv{sideTint: a(.62, .070, .17), wsTint: a(.62, .070, .07), depth: a(.62, .070, .07), chatTint: a(.62, .070, .02)}
		}
	}
	return p
}

// TintSwatch 是设置里色调圆点的颜色(设计稿:oklch(.86, k×.0016, h)),比真正的表面色饱和得多,圆点才看得出区别。
func TintSwatch(hue, k int) ui.Color {
	c := oklchRGB(.86, float64(k)*.0016, float64(hue))
	return ui.RGB(c[0], c[1], c[2])
}

// TintHueRing 返回色相环的若干个色标(设置里「自定义」圆点用):L .85 / C .08 在 0–360° 上等距取点。
func TintHueRing(n int) []ui.GradientStop {
	out := make([]ui.GradientStop, 0, n+1)
	for i := 0; i <= n; i++ {
		c := oklchRGB(.85, .08, float64(i)*360/float64(n))
		out = append(out, ui.GradientStop{Color: ui.RGB(c[0], c[1], c[2]), At: float32(i) / float32(n)})
	}
	return out
}

// oklchRGB 把 OKLCH(L 0–1、C、h 度)换成 sRGB 的 8 位分量(超出色域的分量夹到 0–255)。
func oklchRGB(L, C, hDeg float64) [3]uint8 {
	hr := hDeg * math.Pi / 180
	a, b := C*math.Cos(hr), C*math.Sin(hr)
	l_ := L + 0.3963377774*a + 0.2158037573*b
	m_ := L - 0.1055613458*a - 0.0638541728*b
	s_ := L - 0.0894841775*a - 1.2914855480*b
	l, m, s := l_*l_*l_, m_*m_*m_, s_*s_*s_
	lin := [3]float64{
		4.0767416621*l - 3.3077115913*m + 0.2309699292*s,
		-1.2684380046*l + 2.6097574011*m - 0.3413193965*s,
		-0.0041960863*l - 0.7034186147*m + 1.7076147010*s,
	}
	var out [3]uint8
	for i, v := range lin {
		v = math.Min(math.Max(v, 0), 1)
		if v <= 0.0031308 {
			v *= 12.92
		} else {
			v = 1.055*math.Pow(v, 1/2.4) - 0.055
		}
		out[i] = uint8(math.Round(v * 255))
	}
	return out
}

func oklchHex(L, C, hDeg float64) string {
	c := oklchRGB(L, C, hDeg)
	return fmt.Sprintf("#%02X%02X%02X", c[0], c[1], c[2])
}
