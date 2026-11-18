// Package theme 把设计稿 docs/design/css/tokens.css 的三层令牌一比一搬到 Go:
//
//	① 原始令牌(每套主题一份色板,palettes)
//	② 语义令牌(由原始令牌推导出的 Tokens,界面代码只用这一层)
//	③ 方案令牌(Scheme:圆角 / 面板间隙 / 行高的倍率)
//
// 换主题或方案只需重新 Resolve,组件代码一行都不用动。
package theme

import (
	"github.com/egoist/mygo/ui"
)

// Palette 是一套主题的原始色板。
type Palette struct {
	ID, Name string
	Dark     bool
	// Glass 是液态玻璃主题(流光):面板与浮层画成半透明玻璃,壁纸由 Tokens.Wall 给出(见 glass.go)
	Glass bool
	// 表面与文字
	Base, Surface, Raised, Sunken, Ink string
	// 强调色
	Accent, OnAccent string
	// 状态与装饰色
	Ok, Err, Warn, Info, Vio, Cyan, Pink string
	// 阴影的基色(RGB)与强度倍率:深色主题阴影更重
	Shade  [3]uint8
	KShade float32
}

// Scheme 是布局方案:只调倍率,不碰颜色。
type Scheme struct {
	ID, Name string
	RK       float32 // 圆角倍率
	GapPane  float32 // 面板之间的间隙(浮岛方案才有)
	RowH     float32 // 列表行高
}

// Palettes 是六套主题,顺序与设计稿的主题选择一致。
var Palettes = []Palette{
	{ID: "paper", Name: "宣纸", Base: "#EFEDE8", Surface: "#FAF9F6", Raised: "#FFFFFF", Sunken: "#F3F1EC", Ink: "#1F1D1A",
		Accent: "#D9531E", OnAccent: "#FFFFFF", Ok: "#2E9A5A", Err: "#D2453D", Warn: "#B5781C", Info: "#3A6DD6", Vio: "#7656DC", Cyan: "#0E8C99", Pink: "#C13F8E",
		Shade: [3]uint8{40, 28, 10}, KShade: 1},
	{ID: "mist", Name: "雾霭", Base: "#ECEEF2", Surface: "#F8F9FB", Raised: "#FFFFFF", Sunken: "#F0F2F6", Ink: "#1A1D24",
		Accent: "#4662E8", OnAccent: "#FFFFFF", Ok: "#1F9D62", Err: "#D8434B", Warn: "#B47A14", Info: "#2F7DD8", Vio: "#7B58E0", Cyan: "#0B8FA0", Pink: "#C2418F",
		Shade: [3]uint8{16, 24, 48}, KShade: 1},
	{ID: "celadon", Name: "青瓷", Base: "#E9EDE9", Surface: "#F7F9F6", Raised: "#FFFFFF", Sunken: "#EEF2EE", Ink: "#18201C",
		Accent: "#0E8674", OnAccent: "#FFFFFF", Ok: "#2F9A55", Err: "#CF4A40", Warn: "#A97B16", Info: "#3573C9", Vio: "#7460D0", Cyan: "#0C8C9C", Pink: "#B8478A",
		Shade: [3]uint8{14, 36, 26}, KShade: 1},
	{ID: "ink", Name: "墨夜", Dark: true, Base: "#161513", Surface: "#1C1B19", Raised: "#262522", Sunken: "#181715", Ink: "#EDEAE4",
		Accent: "#FF7A3D", OnAccent: "#1A0E06", Ok: "#4FC283", Err: "#F0665D", Warn: "#E2A942", Info: "#709AF5", Vio: "#A590F7", Cyan: "#4CC3CF", Pink: "#EE7FBF",
		Shade: [3]uint8{0, 0, 0}, KShade: 3.2},
	{ID: "graphite", Name: "石墨", Dark: true, Base: "#121316", Surface: "#17181C", Raised: "#212328", Sunken: "#131418", Ink: "#E8EAEE",
		Accent: "#6F8CFF", OnAccent: "#0A1030", Ok: "#45C08A", Err: "#F2676B", Warn: "#E0AC4A", Info: "#5FA8F5", Vio: "#A48CF8", Cyan: "#45C4D4", Pink: "#EC7DC0",
		Shade: [3]uint8{0, 0, 0}, KShade: 3.2},
	{ID: "aurora", Name: "极光", Dark: true, Base: "#0E1020", Surface: "#13162A", Raised: "#1C2039", Sunken: "#0F1224", Ink: "#E6E8F7",
		Accent: "#A78BFA", OnAccent: "#160F33", Ok: "#5AD1A0", Err: "#FF6F8B", Warn: "#F0B657", Info: "#62B4FF", Vio: "#C29BFF", Cyan: "#5CD6E6", Pink: "#FF86C8",
		Shade: [3]uint8{2, 3, 14}, KShade: 3.6},
}

// Schemes 是三套布局方案。
var Schemes = []Scheme{
	{ID: "a", Name: "墨线", RK: .8, GapPane: 0, RowH: 24},
	{ID: "b", Name: "浮岛", RK: 1.4, GapPane: 6, RowH: 26},
	{ID: "c", Name: "轨道", RK: 1, GapPane: 0, RowH: 22},
}

// 固定尺寸(设计稿 :root),单位 DIP。
const (
	WRail     = 44
	WWs       = 560 // Agent 形态右侧工作区宽度
	WDock     = 352
	WCol      = 712
	IconSize  = 14
	FlowInset = 22
)

// 界面字号与控件高度会随「界面大小」(设置 → 外观)缩放,所以是变量,由 SetUIScale 统一重算。
// 基准值是 100% 时的设计稿尺寸(设计稿 v4 字号阶梯,tokens.css):默认界面大小就是 100%(见 DefaultUIScale)。
// 旧默认 110% 会让界面 14.3 / 阅读 16.5 / 代码 14.3,比同类工具大一级,v4 起改回 100%。
var (
	HCtl    float32 = 24
	HCtlSm  float32 = 20
	HBar    float32 = 36
	HTab    float32 = 30
	HStatus float32 = 22
	WSide   float32 = 228

	// 字号阶梯(设计稿 --fs-*):micro 徽标 / 键帽 · ctl 控件内文字 · cap 说明与附注 · body 界面与工具行 · read 会话正文 · title 卡片标题 · head 页面标题 · display 欢迎页。
	FsMicro   float32 = 10.5
	FsCtl     float32 = 11.5
	FsCap     float32 = 12
	FsBody    float32 = 13
	FsRead    float32 = 14
	FsTitle   float32 = 14
	FsHead    float32 = 17
	FsDisplay float32 = 22
	FsMono    float32 = 12 // 代码一律 12(≈ read × 0.86),与说明字同号
)

// 说明文字(cap)在界面缩到最小时也不低于此值(设计稿 typo-width)。
const minCapPx float32 = 11.5 // 说明字缩到最小时的下限(同 FsCtl 的基准)

// 界面大小的范围与默认值(百分数)。
const (
	MinUIScale     = 90
	MaxUIScale     = 150
	DefaultUIScale = 100
)

var uiScale float32 = 1

// UIScale 返回当前界面缩放倍率(1 = 100%)。
func UIScale() float32 { return uiScale }

// Px 把一个基准尺寸(字号 / 行高 / 图标)按界面缩放换算,取到 0.5 的整倍数,文字边缘才不发虚。
func Px(base float32) float32 {
	v := base * uiScale
	return float32(int(v*2+.5)) / 2
}

// SetUIScale 设置界面大小(百分数,夹在 MinUIScale ~ MaxUIScale 之间)并重算所有缩放后的尺寸。
func SetUIScale(pct int) {
	pct = min(max(pct, MinUIScale), MaxUIScale)
	uiScale = float32(pct) / 100
	HCtl, HCtlSm, HBar = Px(24), Px(20), Px(36)
	HTab, HStatus = Px(30), Px(22)
	WSide = float32(int(228*(1+(uiScale-1)*.8) + .5)) // 侧栏随字号略加宽,但不按全比例,免得挤占对话区
	FsMicro, FsCtl, FsBody, FsRead = Px(10.5), Px(11.5), Px(13), Px(14)
	FsCap = max(Px(12), minCapPx)
	FsTitle, FsHead, FsDisplay = Px(14), Px(17), Px(22)
	FsMono = Px(12)
}

func init() { SetUIScale(DefaultUIScale) }

// 行高令牌(设计稿 --lh-*)。
const (
	LhTight = 1.3
	LhDense = 1.35
	LhUI    = 1.4
	LhBody  = 1.5
	LhCap   = 1.45 // 说明文字常用
	LhChat  = 1.55
	LhMono  = 1.6
	LhRead  = 1.6
)

// 间距刻度 --sp-1 … --sp-8。
const (
	Sp1 = 2
	Sp2 = 4
	Sp3 = 6
	Sp4 = 8
	Sp5 = 12
	Sp6 = 16
	Sp7 = 24
	Sp8 = 32
)

// Tokens 是解析后的语义令牌,界面代码只读它。
type Tokens struct {
	Palette Palette
	Scheme  Scheme

	// 表面
	Chrome, BG, Raised, Sunken ui.Color
	// 文字四级
	Tx1, Tx2, Tx3, Tx4 ui.Color
	// 描边三级
	Bd1, Bd2, Bd3 ui.Color
	// 交互填充:悬停 / 选中 / 按下
	Fill1, Fill2, Fill3 ui.Color
	// 强调色
	Ac, OnAc, AcSoft, AcLine, AcTx ui.Color
	// 列表选中态:会话 / 文件树 / 大纲等所有列表共用,强调色淡染,比 Fill2 的灰底明显得多
	Sel ui.Color
	// 状态色
	Ok, Err, Warn, Info                 ui.Color
	OkSoft, ErrSoft, WarnSoft, InfoSoft ui.Color
	Vio, Cyan, Pink                     ui.Color
	// 工具族:会话流里每类工具一个色相,只染图标
	ToolExplore, ToolEdit, ToolExec, ToolWeb, ToolMCP, ToolAgent, ToolPlan, ToolThink ui.Color
	// 语法色
	SynKw, SynStr, SynFn, SynType, SynNum, SynMac, SynCom ui.Color
	// 差异
	DiffAdd, DiffDel ui.Color
	// ReduceTransp 是系统「降低透明度」/「增强对比度」是否开启(由 app.retheme 写入):开启时玻璃主题的 Tone() 全部换成不透明色,
	// 面板的染色 / 深度 / 高光也都归零(MyGo 的玻璃插件自己会退成不透明的表面色)。
	ReduceTransp bool
	// 遮罩与阴影基色
	Scrim  ui.Color
	shade  [3]uint8
	kShade float32
}

// mix 等价于 CSS 的 color-mix(in srgb, a pct%, b):a 占 pct 份,其余是 b。
func mix(a ui.Color, pct float32, b ui.Color) ui.Color { return b.Mix(a, pct/100) }

// over 等价于 color-mix(in srgb, a pct%, transparent):只改透明度,颜色不变。
func over(a ui.Color, pct float32) ui.Color { return a.Alpha(pct / 100) }

// Find 按 ID 找主题,找不到返回第一套。
func Find(id string) Palette {
	for _, p := range Palettes {
		if p.ID == id {
			return p
		}
	}
	return Palettes[0]
}

// FindScheme 按 ID 找方案,找不到返回第一套。
func FindScheme(id string) Scheme {
	for _, s := range Schemes {
		if s.ID == id {
			return s
		}
	}
	return Schemes[0]
}

// Resolve 由主题与方案推导全部语义令牌(对应 tokens.css 的「② 语义令牌」段)。
// accent 非空时覆盖主题自带的强调色(设置 → 外观里的强调色选择)。
func Resolve(p Palette, s Scheme, accent string) *Tokens {
	h := ui.Hex
	if accent == "" {
		accent = p.Accent
	}
	base, surface, raised, sunken, ink := h(p.Base), h(p.Surface), h(p.Raised), h(p.Sunken), h(p.Ink)
	ac := h(accent)
	t := &Tokens{Palette: p, Scheme: s, shade: p.Shade, kShade: p.KShade}
	t.Chrome, t.BG, t.Raised, t.Sunken = base, surface, raised, sunken

	t.Tx1 = ink
	t.Tx2 = mix(ink, 68, surface)
	t.Tx3 = mix(ink, 46, surface)
	t.Tx4 = mix(ink, 30, surface)
	t.Bd1, t.Bd2, t.Bd3 = over(ink, 7), over(ink, 12), over(ink, 20)
	if p.Glass {
		// 玻璃主题的描边更淡:原型里卡片 / 输入框几乎没有轮廓线,靠柔和的高光边与软阴影「浮」起来;
		// 沿用常规浓度会是一圈灰色硬线,看起来是贴在底上的实色块
		t.Bd1, t.Bd2, t.Bd3 = over(ink, 4), over(ink, 7), over(ink, 13)
	}
	t.Fill1, t.Fill2, t.Fill3 = over(ink, 5), over(ink, 8.5), over(ink, 13)

	t.Ac, t.OnAc = ac, h(p.OnAccent)
	t.AcSoft = mix(ac, 13, raised)
	t.AcLine = mix(ac, 45, surface)
	t.AcTx = mix(ac, 84, ink)
	if p.Dark {
		t.Sel = over(ac, 24)
	} else {
		t.Sel = over(ac, 16)
	}

	t.Ok, t.Err, t.Warn, t.Info = h(p.Ok), h(p.Err), h(p.Warn), h(p.Info)
	t.Vio, t.Cyan, t.Pink = h(p.Vio), h(p.Cyan), h(p.Pink)
	t.OkSoft = mix(t.Ok, 14, raised)
	t.ErrSoft = mix(t.Err, 12, raised)
	t.WarnSoft = mix(t.Warn, 15, raised)
	t.InfoSoft = mix(t.Info, 13, raised)

	t.ToolExplore, t.ToolEdit, t.ToolExec, t.ToolWeb = t.Info, t.Ok, t.Vio, t.Cyan
	t.ToolMCP, t.ToolAgent, t.ToolPlan, t.ToolThink = t.Pink, ac, t.Warn, t.Tx3

	t.SynKw, t.SynStr, t.SynFn, t.SynType = t.Vio, t.Ok, t.Info, t.Warn
	t.SynNum = mix(ac, 60, t.Err)
	t.SynMac, t.SynCom = t.Cyan, t.Tx3

	t.DiffAdd, t.DiffDel = over(t.Ok, 14), over(t.Err, 13)
	t.Scrim = t.shadeColor(.2*p.KShade + .06)
	return t
}

// shadeColor 是阴影基色按不透明度 a 的颜色(对应 rgb(var(--c-shade) / a))。
func (t *Tokens) shadeColor(a float32) ui.Color {
	return ui.RGBA(t.shade[0], t.shade[1], t.shade[2], a)
}

// 圆角:基准值乘方案倍率(--r-xs … --r-xl),--r-full 固定 999。
func (t *Tokens) RXs() float32 { return 3 * t.Scheme.RK }
func (t *Tokens) RSm() float32 { return 5 * t.Scheme.RK }
func (t *Tokens) RMd() float32 { return 7 * t.Scheme.RK }
func (t *Tokens) RLg() float32 { return 10 * t.Scheme.RK }
func (t *Tokens) RXl() float32 { return 14 * t.Scheme.RK }

// RFull 胶囊圆角。
const RFull = 999

// Dark 是否深色主题。
func (t *Tokens) Dark() bool { return t.Palette.Dark }

// Shadow1 / Shadow2 / Shadow3 对应 --sh-1 … --sh-3 的阴影层:每个返回(x, y, blur, spread, 颜色)列表,
// 用法:for _, s := range t.Shadow2() { el.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color) }。
// 设计稿里的 `0 0 0 1px var(--bd)` 描边层不在这里(那是边框,由调用方 Border 画)。
type ShadowLayer struct {
	X, Y, Blur, Spread float32
	Color              ui.Color
}

func (t *Tokens) Shadow1() []ShadowLayer {
	if t.Palette.Glass {
		return t.GlassShadowS() // 卡片 / 选中行 / 输入框 / 滑块:柔和的「抬起」阴影(设计稿 --glass-sh-s)
	}
	return []ShadowLayer{{0, 1, 2, 0, t.shadeColor(.06 * t.kShade)}}
}

// ShadowStuck 吸顶卡的阴影(--sh-stuck):只在卡片正下方一小道,负扩散收在两侧之内,
// 不会像 Shadow2 那样在整行下面拖出长暗带。
func (t *Tokens) ShadowStuck() []ShadowLayer {
	return []ShadowLayer{{0, 3, 8, -4, t.shadeColor(.22 * t.kShade)}}
}

func (t *Tokens) Shadow2() []ShadowLayer {
	if t.Palette.Glass {
		return t.GlassShadow() // 玻璃主题:浮层统一用玻璃岛的悬浮阴影(夜色下不乘 KShade,免得一圈黑)
	}
	return []ShadowLayer{
		{0, 10, 30, -6, t.shadeColor(.18 * t.kShade)},
		{0, 2, 6, 0, t.shadeColor(.07 * t.kShade)},
	}
}

func (t *Tokens) Shadow3() []ShadowLayer {
	if t.Palette.Glass {
		return t.GlassShadow()
	}
	return []ShadowLayer{{0, 28, 70, -12, t.shadeColor(.3 * t.kShade)}}
}

// Ring 是聚焦环的颜色(--ring:强调色 18%,宽 3px)。
func (t *Tokens) Ring() ui.Color { return over(t.Ac, 18) }

// UITheme 把令牌转成 mygo 的 ui.Theme,让内置控件(输入框、滚动条、开关……)与自绘部分同色。
func (t *Tokens) UITheme() *ui.Theme {
	var th ui.Theme
	if t.Palette.Dark {
		th = *ui.DarkTheme()
	} else {
		th = *ui.LightTheme()
	}
	th.Background = t.BG
	th.Surface = t.Raised
	th.SurfaceHover = t.Fill2
	th.SurfacePressed = t.Fill3
	th.Border = t.Bd2
	th.Text = t.Tx1
	th.TextMuted = t.Tx3
	th.Accent = t.Ac
	th.AccentHover = t.Ac.Mix(t.Tx1, .08)
	th.AccentPressed = t.Ac.Mix(t.Tx1, .16)
	th.AccentText = t.OnAc
	th.InactiveAccent = t.Tx3
	th.Danger, th.Warning, th.Success = t.Err, t.Warn, t.Ok
	th.Selection = over(t.Ac, 28)
	th.Focus = t.Ac
	th.Radius = t.RSm()
	th.Spacing = 4
	th.FontSize = FsBody // 没写字号的文字 = 界面正文 13(会话正文显式用 FsRead)
	th.Scrollbar = t.Bd3
	th.ScrollbarWidth = 6
	th.Inverse, th.InverseText = t.Tx1, t.BG
	return &th
}
