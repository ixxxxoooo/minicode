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

// 面板通透度(glass.Glass.Clarity):取插件按 macOS 实测的材质,不自己调。
//   - ClarityContent = 0:macOS 的标准材质(最磨砂),对话栏 / 整页 / 浮层这类文字多的面板;
//   - ClaritySidebar = 0.5:插件的「中档」,模糊减半、提亮减弱,即 macOS 26 浮动侧栏那种能透出壁纸的通透度。
const (
	ClarityContent float32 = 0
	ClaritySidebar float32 = 0.5
)

// GlassMat 返回面板用的玻璃材质。clarity:0 = 最磨砂(对话 / 整页 / 浮层,文字多),
// 约 0.35 = 更通透(导航栏、工作区)。系统「降低透明度」时插件自己退成不透明的表面色。
func GlassMat(clarity float32) ui.Material { return glass.Glass{Clarity: clarity} }

// GlassPane 是带暖色深浅的面板玻璃:tint 取自 Tokens.Tone() 的 SideTint / WsTint。
// 壁纸是纯色,面板背后没有内容可透,层次只能靠这层染色(侧栏最深、工作区居中、对话栏最亮)。
func GlassPane(clarity float32, tint ui.Color) ui.Material {
	return glass.Glass{Clarity: clarity, Tint: tint}
}

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

// glassBtn 是玻璃主题下默认按钮:真玻璃(可交互:悬停亮边、按下微胀)。
// 同屏文字按钮只有几个到十几个、面积很小,每块只抓自己周围一小块背后画面,代价可以接受;性能实测见设计稿 B9。
func glassBtn(b ui.Element, _ bool) {
	b.Material(GlassControl())
	for _, s := range T.Shadow1() {
		b.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	GlassHi(b)
}

// glassPri 是主要按钮:着色玻璃(强调色几乎不透明,只透出一点)+ 强调色光晕(设计稿 --ac-glow:整页唯一「发光」的地方)。
func glassPri(b ui.Element) {
	// 着色取满(不透明):插件会把着色按 88% 叠上并提亮底下的内容,取 .92 时陶土橙发粉、白色箭头对比不够;取满后颜色饱满,仍保留玻璃的边缘高光与按压反馈
	b.Material(GlassTinted(T.Ac)).Shadow(0, 6, 14, -6, T.Ac.Alpha(.55))
}

// GlassAccent 把一个强调色实心的按钮(发送键)改成着色玻璃 + 强调色光晕,与主要按钮同一套(设计稿 .ib.send / .btn.pri)。
// 调用方先照常设好 Background / 圆角 / 文字色;玻璃主题下这里用 Material 盖掉底色,悬停亮边、按下微胀由插件的 Interactive 负责。
// 非玻璃主题原样返回。参数 b:按钮元素。返回 b。
func GlassAccent(b ui.Element) ui.Element {
	if T.Glass() {
		glassPri(b)
	}
	return b
}

// GlassRim 给玻璃面板画一圈渐变高光边:左上亮、右下暗,像光沿着玻璃边缘走(设计稿 .pane::after)。
// 插件自带的边缘高光只有 1px 且很弱,在淡雅的壁纸上几乎看不见,这里补一圈。
func GlassRim(el ui.Element) ui.Element {
	if !T.Glass() {
		return el
	}
	tn := T.Tone()
	return el.Border(1, tn.RimLo).BorderLinearGradient(ui.LinearGradient{Angle: 145, Stops: []ui.GradientStop{
		{Color: tn.Rim, At: 0}, {Color: tn.RimLo, At: .36}, {Color: tn.RimLo.Alpha(0), At: .6}, {Color: tn.RimLo, At: 1},
	}})
}

// FloatLift 是 Float 加一层轻阴影:给浮在会话流之上的输入区部件(输入框、待办条、排队卡、计划卡)用,
// 让它们从内容上「抬起来」。阴影取最小一档(Shadow1),玻璃自己的高光边负责勾轮廓。
func FloatLift(el ui.Element) ui.Element {
	if !T.Glass() {
		return el
	}
	el.Material(GlassMat(0)).BorderColor(T.Tone().RimLo) // 描边换成玻璃的浅色高光边,不要灰色硬线
	for _, s := range T.Shadow1() {
		el.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	return GlassHi(el)
}

// GlassPaintClear 在 Draw 回调里画一块清透玻璃(插件的 Clear 样式:几乎不模糊、只有边缘折射与高光),
// 带抬起的软阴影:给迷你地图的视口滑块这类「要透出底下内容、又要有立体感」的小块用。
// 参数 p / r:Draw 给的画笔与盒子;radius:圆角。降低透明度时插件自己退成不透明表面。
func GlassPaintClear(p *ui.Painter, r ui.Rect, radius float32) {
	for _, s := range T.GlassShadowS() {
		p.Shadow(r, radius, s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	glass.Paint(p, r, radius, glass.Glass{Style: glass.Clear})
}

// GlassHi 给元素顶部加 1px 的内高光(设计稿 inset 0 1px 0 --glass-hi):按钮、卡片、输入框的「厚度」感。
// 非玻璃主题 / 降低透明度时 Tone().Hi 是透明的,等于什么都不画。
func GlassHi(el ui.Element) ui.Element {
	if !T.Glass() {
		return el
	}
	return el.InsetShadow(0, 1, 0, 0, T.Tone().Hi)
}

// Lift 让面板上的卡片「浮」起来:软阴影 + 顶部内高光 + 玻璃卡片底色(设计稿 .card)。只在玻璃主题生效。
// 参数 el:已设好圆角与描边的卡片元素。返回 el。
func Lift(el ui.Element) ui.Element {
	if !T.Glass() {
		return el
	}
	el.Background(RaisedBG())
	for _, s := range T.Shadow1() {
		el.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	return GlassHi(el)
}

// GlassSheen 给玻璃面板顶部叠一层渐隐的白色高光(设计稿 --glass-sheen:顶部 26% 高度的白色渐变),
// 让玻璃有「厚度」而不是平的半透明。画在面板底(材质)之后、内容之前;高度取面板的 26%,最高 220。
//
// 参数 radius 必须是该元素的圆角:Draw 里的绘制不会自动按元素圆角裁剪,不裁就会在四角外面露出方角的亮块
// (方案 B 的圆角岛上最明显),所以整段高光放进 p.Clip(盒子, radius) 里画。
func GlassSheen(el ui.Element, radius float32) ui.Element {
	if !T.Glass() {
		return el
	}
	tn := T.Tone()
	c, depth := tn.Sheen, tn.Depth
	return el.Draw(func(p *ui.Painter, r ui.Rect) {
		p.Clip(r, radius, func() {
			// 底部的暖色「深度」:顶部 35% 以上为 0,往下渐强到底(原型里面板从顶到底逐渐变深)
			p.FillGradient(r, ui.LinearGradient{Angle: 180, Stops: []ui.GradientStop{
				{Color: depth.Alpha(0), At: .35}, {Color: depth, At: 1},
			}}, 0)
			// 顶部的白色高光
			h := min(r.H*.26, 220)
			p.FillGradient(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: h}, ui.LinearGradient{Angle: 180, From: c, To: c.Alpha(0)}, 0)
		})
	})
}

// GlassTinted 返回带着色的玻璃(主要按钮、发送键):tint 的透明度决定染得多重。
func GlassTinted(tint ui.Color) ui.Material { return glass.Glass{Tint: tint, Interactive: true} }

// GlassControl 返回可交互的小块玻璃(悬停亮边、按下微胀),给标题栏搜索胶囊、分段滑块这类少量大控件用。
func GlassControl() ui.Material { return glass.Glass{Interactive: true} }
