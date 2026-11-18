package theme

import "github.com/egoist/mygo/ui"

// 流光(液态玻璃)主题:对应设计稿 css/tokens.css 的 glow / glow-night 与 css/glass.css。
// 色板与其余六套同构(13 色);玻璃的模糊 / 折射 / 高光由 MyGo 的 glass 插件决定,
// 这里只给出壁纸与「玻璃之上」要用到的几个半透明色。
//
// @author ixxxxoooo

// glassPalettes 是追加在 Palettes 末尾的两套玻璃主题(追加而不是插入,偏好里按序号存的「应用图标配色」不会错位)。
var glassPalettes = []Palette{
	{ID: "glow", Name: "流光", Glass: true, Base: "#E9EEF8", Surface: "#F7F9FE", Raised: "#FFFFFF", Sunken: "#EDF1F9", Ink: "#16192B",
		Accent: "#4A6FD3", OnAccent: "#FFFFFF", Ok: "#1F9D62", Err: "#D8434B", Warn: "#B47A14", Info: "#2F7DD8", Vio: "#7B58E0", Cyan: "#0B8FA0", Pink: "#C2418F",
		Shade: [3]uint8{30, 40, 90}, KShade: 1},
	{ID: "glow-night", Name: "流光·夜", Dark: true, Glass: true, Base: "#0B0E1C", Surface: "#12162B", Raised: "#1B2040", Sunken: "#0D1022", Ink: "#EAEEFF",
		Accent: "#9DB8F2", OnAccent: "#0A1430", Ok: "#5AD1A0", Err: "#FF6F8B", Warn: "#F0B657", Info: "#62B4FF", Vio: "#B79BFF", Cyan: "#5CD6E6", Pink: "#FF86C8",
		Shade: [3]uint8{2, 3, 14}, KShade: 3.6},
}

func init() { Palettes = append(Palettes, glassPalettes...) }

// Glass 当前是否玻璃主题。界面代码只用它决定「画玻璃还是画实色」,其余令牌照常取。
func (t *Tokens) Glass() bool { return t.Palette.Glass }

// WallBlob 是壁纸上的一团色晕:圆心(占窗口宽高的比例)、半径(占窗口宽高均值的比例)、颜色(中心色,边缘渐隐到同色全透明)。
type WallBlob struct {
	X, Y, K float32
	Color   ui.Color
}

// WallSpec 是壁纸:纵向两色底、几团低饱和色晕、一道斜向高光带(Streak)。
// 高光带给玻璃的折射一条明暗边缘——背后全是柔和渐变时,透镜效果看起来就和一层白膜没区别。
type WallSpec struct {
	Top, Bottom ui.Color
	Blobs       []WallBlob
	Streak      ui.Color // 高光带中心色(两侧渐隐到同色全透明)
}

// Wall 返回壁纸。静态,整窗只画一次(设计稿 --wall:低饱和,淡雅)。
func (t *Tokens) Wall() WallSpec {
	h := ui.Hex
	if t.Palette.Dark {
		return WallSpec{Top: h("#0E1226"), Bottom: h("#090C1A"), Streak: ui.RGBA(190, 200, 255, .08), Blobs: []WallBlob{
			{.10, .08, .34, ui.RGBA(78, 86, 190, .42)}, {.94, .10, .30, ui.RGBA(150, 80, 150, .26)},
			{.80, 1.00, .38, ui.RGBA(40, 120, 140, .32)}, {.04, .96, .29, ui.RGBA(70, 90, 170, .30)},
		}}
	}
	return WallSpec{Top: h("#F1F3F8"), Bottom: h("#E6EAF3"), Streak: ui.RGBA(255, 255, 255, .55), Blobs: []WallBlob{
		{.10, .08, .34, h("#C2D2F2")}, {.94, .10, .30, h("#EBCFE0")},
		{.80, 1.00, .38, h("#C5E3E8")}, {.04, .96, .29, h("#D6CCEE")},
		{.52, .52, .20, h("#F3E4CF")},
	}}
}

// GlassShadow 是玻璃岛的悬浮阴影(设计稿 --glass-sh):不乘 KShade,夜色下也只是一圈很淡的深蓝黑。
func (t *Tokens) GlassShadow() []ShadowLayer {
	return []ShadowLayer{{0, 14, 36, -12, t.shadeColor(.24)}, {0, 2, 8, 0, t.shadeColor(.07)}}
}

// GlassTone 是玻璃之上的几个半透明色(设计稿 --glass-3 / --glass-card / --glass-sel / 边缘高光)。
type GlassTone struct {
	Thin  ui.Color // 薄玻璃:按钮、输入框、分段槽
	Card  ui.Color // 面板里的卡片
	Sel   ui.Color // 选中行、分段滑块、悬停
	Rim   ui.Color // 边缘高光(亮边)
	RimLo ui.Color // 边缘高光(暗边)
	Sheen ui.Color // 顶部光泽起始色
	Mid   ui.Color // 玻璃面板之上的轻染色(弹层底衬,保证文字对比)
}

// Tone 返回当前主题的玻璃半透明色;非玻璃主题返回由常规令牌推出的不透明近似,调用方可以直接用。
func (t *Tokens) Tone() GlassTone {
	if !t.Palette.Glass {
		return GlassTone{Thin: t.Fill1, Card: t.Raised, Sel: t.Raised, Rim: t.Bd1, RimLo: t.Bd1, Sheen: ui.Transparent, Mid: t.BG}
	}
	if t.Palette.Dark {
		return GlassTone{Thin: ui.RGBA(150, 168, 255, .10), Card: ui.RGBA(40, 50, 108, .46), Sel: ui.RGBA(150, 168, 255, .20),
			Rim: ui.RGBA(200, 214, 255, .34), RimLo: ui.RGBA(200, 214, 255, .10), Sheen: ui.RGBA(190, 205, 255, .16), Mid: ui.RGBA(22, 28, 66, .60)}
	}
	return GlassTone{Thin: ui.RGBA(255, 255, 255, .30), Card: ui.RGBA(255, 255, 255, .72), Sel: ui.RGBA(255, 255, 255, .78),
		Rim: ui.RGBA(255, 255, 255, .85), RimLo: ui.RGBA(255, 255, 255, .38), Sheen: ui.RGBA(255, 255, 255, .55), Mid: ui.RGBA(255, 255, 255, .64)}
}
