package theme

import "github.com/egoist/mygo/ui"

// 流光(液态玻璃)主题:对应设计稿 css/tokens.css 的 glow / glow-night 与 css/glass.css。
// 色板与其余六套同构(13 色);玻璃的模糊 / 折射 / 高光由 MyGo 的 glass 插件决定,
// 这里只给出壁纸与「玻璃之上」要用到的几个半透明色。
//
// @author ixxxxoooo

// glassPalettes 是追加在 Palettes 末尾的两套玻璃主题(追加而不是插入,偏好里按序号存的「应用图标配色」不会错位)。
var glassPalettes = []Palette{
	// 配色取 Claude 品牌色:Ivory #FAF9F5 / Slate #141413 / Clay 陶土橙(#D97757,浅色下压深一档,白字对比 4.5:1)。壁纸不用任何彩色。
	{ID: "glow", Name: "流光", Glass: true, Base: "#F0EEE6", Surface: "#FAF9F5", Raised: "#FFFFFF", Sunken: "#F3F1EA", Ink: "#141413",
		Accent: "#BD5A36", OnAccent: "#FFFFFF", Ok: "#2E9A5A", Err: "#D2453D", Warn: "#B5781C", Info: "#3A6DD6", Vio: "#7656DC", Cyan: "#0E8C99", Pink: "#C13F8E",
		Shade: [3]uint8{40, 28, 10}, KShade: 1},
	{ID: "glow-night", Name: "流光·夜", Dark: true, Glass: true, Base: "#131211", Surface: "#1A1917", Raised: "#262320", Sunken: "#151413", Ink: "#F0EEE6",
		Accent: "#E8967A", OnAccent: "#2A1208", Ok: "#4FC283", Err: "#F0665D", Warn: "#E2A942", Info: "#709AF5", Vio: "#A590F7", Cyan: "#4CC3CF", Pink: "#EE7FBF",
		Shade: [3]uint8{0, 0, 0}, KShade: 3.2},
}

func init() { Palettes = append(Palettes, glassPalettes...) }

// Glass 当前是否玻璃主题。界面代码只用它决定「画玻璃还是画实色」,其余令牌照常取。
func (t *Tokens) Glass() bool { return t.Palette.Glass }

// 流光的壁纸是纯色:窗口底色就是 Tokens.Chrome(浅 #F0EEE6 / 夜 #131211),没有渐变、色晕或高光带。

// GlassShadow 是玻璃岛的悬浮阴影(设计稿 --glass-sh):不乘 KShade,夜色下也只是一圈很淡的暗影。
func (t *Tokens) GlassShadow() []ShadowLayer {
	return []ShadowLayer{{0, 14, 36, -12, t.shadeColor(.24)}, {0, 2, 8, 0, t.shadeColor(.07)}}
}

// GlassShadowS 是小块玻璃 / 卡片的抬起阴影(设计稿 --glass-sh-s:0 6 16 -8 / 0 1 3):整体偏下的一圈软影,让元素像浮在面板上。
func (t *Tokens) GlassShadowS() []ShadowLayer {
	return []ShadowLayer{{0, 6, 16, -8, t.shadeColor(.22)}, {0, 1, 3, 0, t.shadeColor(.08)}}
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
	// 面板的暖色深浅(玻璃的 Tint):壁纸是纯色,面板之间的明暗层次靠它——侧栏最深、工作区居中、对话栏最亮(不染)
	SideTint, WsTint ui.Color
	// Depth 是面板底部叠的暖色(纵向渐变的终点,顶部为 0):原型里面板从顶到底逐渐变深变暖
	// (对话栏实测 252 → 248 → 243),没有它整块面板是一个平的白,看起来「一样」、没有厚度;
	// ChatTint 是对话栏 / 整页内容面板整体的暖色(比侧栏、工作区淡得多),让内容面板不是纯白。
	Depth, ChatTint ui.Color
	// Hair 是不透明的细线色(≈ 原版主题里 --bd-2 画出的浅线):方案 A / C 面板相接处用它盖住玻璃着色器在形状边缘留下的 1px 暗边
	Hair ui.Color
	// Hi 是元素顶部 1px 的内高光(设计稿 inset 0 1px 0 --glass-hi):按钮、卡片、输入框的「厚度」感
	Hi ui.Color
}

// Tone 返回当前主题的玻璃半透明色;非玻璃主题返回由常规令牌推出的不透明近似,调用方可以直接用。
// 系统开了「降低透明度」(Tokens.ReduceTransp)时,玻璃主题也返回同样的不透明近似:染色 / 深度 / 高光全部归零。
func (t *Tokens) Tone() GlassTone {
	if !t.Palette.Glass || t.ReduceTransp {
		return GlassTone{Thin: t.Fill1, Card: t.Raised, Sel: t.Raised, Rim: t.Bd1, RimLo: t.Bd1, Sheen: ui.Transparent, Mid: t.BG, SideTint: ui.Transparent, WsTint: ui.Transparent, Depth: ui.Transparent, ChatTint: ui.Transparent, Hair: t.Bd2, Hi: ui.Transparent}
	}
	g := t.toneBase()
	if ov := t.Palette.glass; ov != nil { // 表面色调:暖色染色 / 高光换成同色相(见 tint.go)
		if t.Palette.Dark {
			g.Thin, g.Card, g.Sel, g.Rim, g.RimLo, g.Sheen, g.Mid, g.Hi = ov.thin, ov.card, ov.sel, ov.rim, ov.rimLo, ov.sheen, ov.mid, ov.hi
		} else {
			g.SideTint, g.WsTint, g.Depth, g.ChatTint = ov.sideTint, ov.wsTint, ov.depth, ov.chatTint
		}
	}
	return g
}

// toneBase 是流光主题默认(没套表面色调)的玻璃半透明色。
func (t *Tokens) toneBase() GlassTone {
	if t.Palette.Dark {
		return GlassTone{Thin: ui.RGBA(255, 235, 220, .08), Card: ui.RGBA(58, 50, 44, .46), Sel: ui.RGBA(255, 235, 220, .16),
			Rim: ui.RGBA(255, 240, 230, .30), RimLo: ui.RGBA(255, 240, 230, .09), Sheen: ui.RGBA(255, 236, 222, .14), Mid: ui.RGBA(38, 34, 30, .60),
			SideTint: ui.RGBA(0, 0, 0, .36), WsTint: ui.RGBA(0, 0, 0, .20), Depth: ui.RGBA(0, 0, 0, .16), ChatTint: ui.RGBA(0, 0, 0, .06), Hair: t.Chrome.Mix(t.Tx1, .12),
			Hi: ui.RGBA(255, 244, 236, .16)}
	}
	return GlassTone{Thin: ui.RGBA(255, 255, 255, .30), Card: ui.RGBA(255, 255, 255, .72), Sel: ui.RGBA(255, 255, 255, .78),
		Rim: ui.RGBA(255, 255, 255, .85), RimLo: ui.RGBA(255, 255, 255, .38), Sheen: ui.RGBA(255, 255, 255, .55), Mid: ui.RGBA(255, 255, 255, .64),
		SideTint: ui.RGBA(150, 120, 90, .17), WsTint: ui.RGBA(150, 120, 90, .07), Depth: ui.RGBA(150, 120, 90, .07), ChatTint: ui.RGBA(150, 120, 90, .02), Hair: t.Chrome.Mix(t.Tx1, .10),
		Hi: ui.RGBA(255, 255, 255, .95)}
}
