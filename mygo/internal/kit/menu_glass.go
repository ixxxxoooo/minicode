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

// glassMenuItem 给菜单里「被高亮」的项套上「更亮的选中玻璃」。框架把高亮项(鼠标移上去或键盘上下选中)涂成实色强调色底(陶土橙块),
// 在玻璃面板上很突兀;这里在框架的样式之后只覆盖背景:换成选中玻璃色 + 顶部内高光。
// 必须按框架的 Highlighted() 判断,而不是 Hovered():两者不同步时(键盘选中、指针移出后高亮还在、动画中途)会漏出橙色块或出现两行同时有底。
// 文字 / 图标色不动:th.AccentText 已取正文色,危险项的红字由框架自己处理。
func glassMenuItem(item ui.Element) {
	if !item.Highlighted() {
		return
	}
	item.Background(T.Tone().Sel)
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

// MenuBtnOpen 同 MenuBtn,但把菜单的开合写进 *open,供「悬停才显示」的按钮在菜单打开期间保持存在:
// 菜单一弹出,指针下的元素就成了浮层,宿主行的 Hovered() 变 false,按钮若随之被拆掉,菜单也跟着消失。
// 参数 el:按钮元素;open:菜单是否正显示(框架在开合时写入);build:构建菜单项的函数。返回 el。
func MenuBtnOpen(el ui.Element, open *bool, build func(m *ui.Menu)) ui.Element {
	opt := ui.PopoverMenuButtonOptions{Open: open}
	if T.Glass() {
		opt.Panel, opt.Item = glassMenuPanel, glassMenuItem
	}
	return el.PopoverMenuButton(build, opt)
}
