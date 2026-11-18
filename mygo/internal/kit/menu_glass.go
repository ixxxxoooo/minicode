package kit

import "github.com/egoist/mygo/ui"

// 右键菜单 / 「···」菜单的玻璃面板。
//
// MiniCode 让窗口自己画菜单(App.View 里 SetMenuPresentation(ui.PopoverMenus)),外观取 ui.Theme:
// 面板是不透明的 Background + Border。框架只允许通过 PopoverMenu / PopoverMenuButton 的选项(Panel)
// 定制面板材质,ContextMenu / Menu 没有选项,所以这里提供两个包装:玻璃主题改走 PopoverMenu 并给面板套玻璃,
// 其余主题原样调用 ContextMenu / Menu,行为不变。
//
// @author ixxxxoooo

// glassMenuPanel 给弹出菜单的面板(含子菜单)套上玻璃:磨砂材质 + 浅色高光边,阴影沿用框架自带的。
func glassMenuPanel(panel ui.Element) {
	panel.Material(GlassMat(0)).BorderColor(T.Tone().RimLo)
	GlassSheen(panel, T.RSm()+2) // 框架里菜单面板的圆角 = Theme.Radius + 2,Theme.Radius 取的是 RSm
}

// glassMenuItem 给菜单项的悬停态套上「更亮的选中玻璃」。框架画的菜单把悬停 / 键盘高亮的项涂成实色强调色底(陶土橙块 + 浅色字),
// 在玻璃面板上很突兀;这里在框架的样式之后覆盖:悬停时换成选中玻璃色 + 顶部内高光 + 正文色。
// 局限:Item 回调拿不到「键盘高亮」状态,只能处理鼠标悬停;键盘上下选中的项仍是框架的强调色底;
// 悬停到危险项时文字也会变回正文色(框架在高亮时才改色,这里无法分辨危险项)。
func glassMenuItem(item ui.Element) {
	if !item.Hovered() {
		return
	}
	item.Background(T.Tone().Sel).TextColor(T.Tx1)
	GlassHi(item)
}

// CtxMenu 给元素挂右键菜单(等价于 el.ContextMenu(build)):玻璃主题下面板是玻璃。
// 参数 el:挂菜单的元素;build:构建菜单项的函数。返回 el。
func CtxMenu(el ui.Element, build func(m *ui.Menu)) ui.Element {
	if !T.Glass() {
		return el.ContextMenu(build)
	}
	return el.PopoverMenu(build, ui.PopoverMenuOptions{Panel: glassMenuPanel, Item: glassMenuItem})
}

// MenuBtn 把元素变成菜单按钮(等价于 el.Menu(build)):玻璃主题下菜单面板是玻璃。
// 参数 el:按钮元素;build:构建菜单项的函数。返回 el。
func MenuBtn(el ui.Element, build func(m *ui.Menu)) ui.Element {
	if !T.Glass() {
		return el.Menu(build)
	}
	return el.PopoverMenuButton(build, ui.PopoverMenuButtonOptions{Panel: glassMenuPanel, Item: glassMenuItem})
}
