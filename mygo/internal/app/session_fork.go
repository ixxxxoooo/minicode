package app

import (
	"net/url"
)

// SetSessionForker 注入分叉会话能力(nil = 未接,菜单置灰)。
func (a *App) SetSessionForker(f func(fromID string) (newID string, err error)) {
	a.sessFork = f
}

// forkSession 从当前会话分叉出新会话(需后端支持)。
func (a *App) forkSession(from string) {
	if a.sessFork == nil {
		a.toast = "分叉会话后端尚未接入(internal/claude/fork.go)"
		return
	}
	a.bg("fork-session", func() func() {
		newID, err := a.sessFork(from)
		return func() {
			if err != nil {
				log.Warn("分叉会话失败", "err", err)
				a.toast = "分叉失败:" + err.Error()
				return
			}
			log.Info("分叉会话", "from", from, "to", newID)
			a.OpenSession(newID)
		}
	})
}

func (a *App) forkSessionAsk(from string) {
	if from == "" || a.sessFork == nil {
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
	u := url.URL{Scheme: "minicode", Host: "session", Path: id}
	a.copyOut(u.String(), "会话链接")
}
