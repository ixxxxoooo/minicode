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

// CtxMenu 给元素挂右键菜单(等价于 el.ContextMenu(build)):玻璃主题下面板是玻璃。
// 参数 el:挂菜单的元素;build:构建菜单项的函数。返回 el。
func CtxMenu(el ui.Element, build func(m *ui.Menu)) ui.Element {
	if !T.Glass() {
		return el.ContextMenu(build)
	}
	return el.PopoverMenu(build, ui.PopoverMenuOptions{Panel: glassMenuPanel})
}

// MenuBtn 把元素变成菜单按钮(等价于 el.Menu(build)):玻璃主题下菜单面板是玻璃。
// 参数 el:按钮元素;build:构建菜单项的函数。返回 el。
func MenuBtn(el ui.Element, build func(m *ui.Menu)) ui.Element {
	if !T.Glass() {
		return el.Menu(build)
	}
	return el.PopoverMenuButton(build, ui.PopoverMenuButtonOptions{Panel: glassMenuPanel})
}
