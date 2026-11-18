package kit

import (
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
)

// 流光(液态玻璃)主题的材质入口:界面代码只调这里,不直接依赖 glass 插件,
// 以后要换实现(例如给低配机器一个「简化玻璃」)只改这一个文件。
//
// 分工(与设计稿 glow-spec 一致):
//   - 大面积、背后固定只有壁纸的面板 / 浮层 / 输入框:用插件的真玻璃(模糊 + 折射 + 高光边);
//   - 按钮、选中行、输入框这类小控件:用 theme.GlassTone 的半透明色 + 描边,不走着色器
//     (每块真玻璃都要抓一次背后画面,小控件成十上百,代价不划算)。
//
// @author ixxxxoooo

// GlassMat 返回面板用的玻璃材质。clarity:0 = 最磨砂(对话 / 整页 / 浮层,文字多),
// 约 0.35 = 更通透(导航栏、工作区)。系统「降低透明度」时插件自己退成不透明的表面色。
func GlassMat(clarity float32) ui.Material { return glass.Glass{Clarity: clarity} }

// Float 把浮在最上层的面板(命令面板、对话框、弹层、抽屉)画成玻璃,覆盖它原来的底色;阴影由 Shadow2 / Shadow3 在玻璃主题下自动换成玻璃岛阴影。
// 非玻璃主题原样返回。调用方在设好圆角、描边、阴影之后调用即可。
func Float(el ui.Element) ui.Element {
	if T.Glass() {
		el.Material(GlassMat(0))
	}
	return el
}

// RaisedBG 是「抬起一层」的表面色:卡片、输入框、默认按钮。玻璃主题下是半透明的卡片色,其余主题就是 T.Raised。
func RaisedBG() ui.Color {
	if T.Glass() {
		return T.Tone().Card
	}
	return T.Raised
}

// SelBG 是选中行 / 分段滑块的底色:玻璃主题下是更亮的半透明「选中」色(不再用强调色淡染),其余主题是 T.Raised。
func SelBG() ui.Color {
	if T.Glass() {
		return T.Tone().Sel
	}
	return T.Raised
}

// glassBtn 是玻璃主题下默认按钮的外观:半透明卡片色 + 暗边;悬停升为「选中」色。
func glassBtn(b ui.Element, hov bool) {
	tn := T.Tone()
	bg := tn.Card
	if hov {
		bg = tn.Sel
	}
	b.Background(bg).Border(1, tn.RimLo)
	for _, s := range T.Shadow1() {
		b.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
}

// glassPri 给主要按钮加一圈高光描边与强调色光晕(设计稿 --ac-glow:整页唯一「发光」的地方)。
func glassPri(b ui.Element) {
	b.Border(1, ui.RGBA(255, 255, 255, .22)).Shadow(0, 6, 14, -6, T.Ac.Alpha(.7))
}

// GlassTinted 返回带着色的玻璃(主要按钮、发送键):tint 的透明度决定染得多重。
func GlassTinted(tint ui.Color) ui.Material { return glass.Glass{Tint: tint, Interactive: true} }

// GlassControl 返回可交互的小块玻璃(悬停亮边、按下微胀),给标题栏搜索胶囊、分段滑块这类少量大控件用。
func GlassControl() ui.Material { return glass.Glass{Interactive: true} }
