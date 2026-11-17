package app

import "strings"

// @author ixxxxoooo

// onFullPage 当前是不是整页功能(自动化 / 画布 / 浏览器 / 自定义 / 用量);设置页有自己的出口,不算。
func (a *App) onFullPage() bool {
	if a.screen == ScreenUsage {
		return true
	}
	for _, p := range pages {
		if a.screen == p.screen {
			return true
		}
	}
	return false
}

// workContextKey 是工作台此刻的「在看什么」:会话、项目、形态、左栏分段、当前编辑器标签。
// 新会话的临时 id(空或 new:)归一成同一个值,握手后换成真实 id 不算用户换了东西。
func (a *App) workContextKey() string {
	cs := a.curSess
	if cs == "" || strings.HasPrefix(cs, "new:") {
		cs = "~"
	}
	tab, tab2 := "", ""
	if a.ide != nil {
		tab, tab2 = a.ide.active, a.ide.active2
	}
	return cs + "|" + a.proj + "|" + string(rune('0'+int(a.mode))) + "|" + a.nav + "|" + tab + "|" + tab2
}

// pageAutoLeave 每帧调用:停在整页功能(用量等)时,只要用户去操作了别的东西(点会话 / 文件 / 项目、切形态或左栏分段、
// 开 / 切编辑器标签……),工作台上下文就变了,自动退出整页回到工作台,而不是让整页卡在那里、操作看不见效果。
// 进入整页的那一帧只记下当时的上下文。
func (a *App) pageAutoLeave() {
	if !a.onFullPage() {
		a.pageKey = ""
		return
	}
	key := a.workContextKey()
	if a.pageKey == "" {
		a.pageKey = key
		return
	}
	if key != a.pageKey {
		log.Info("操作了别的内容,退出整页", "page", string(a.screen))
		a.leavePage()
		a.pageKey = ""
	}
}
