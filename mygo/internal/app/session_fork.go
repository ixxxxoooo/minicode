package app

import (
	"strings"
)

// forkSession 从指定会话分叉出新会话(对话栏头部菜单);逻辑与回合操作条「分叉」一致,复制全量历史。
func (a *App) forkSession(from string) {
	if from == "" || strings.HasPrefix(from, "new:") {
		return
	}
	if from != a.curSess {
		a.OpenSession(from)
	}
	a.forkTurn("")
}

func (a *App) forkSessionAsk(from string) {
	if from == "" || strings.HasPrefix(from, "new:") {
		return
	}
	a.confirmDlg("分叉会话", "会以当前会话为起点创建一条新会话,历史会复制过去。", "分叉", false, func() {
		a.forkSession(from)
	})
}

func (a *App) copySessionLink(id string) {
	if id == "" {
		return
	}
	u := "minicode://session/" + id
	a.copyOut(u, "会话链接")
}
