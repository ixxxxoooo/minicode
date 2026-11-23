package app

import (
	"fmt"
	"time"
)

// lspOnCrash 语言服务器进程退出后的自动重启(指数退避,最多 3 次)。
func (a *App) lspOnCrash(key string, s *lspSess) {
	hub := &a.lspH
	if hub.sess[key] != s {
		return
	}
	if s.restarts >= 3 {
		s.state, s.err = lspFailed, "连续崩溃,已停止自动重启"
		a.outAppend(outLSP, s.spec.Name+" 已停止:查看日志")
		return
	}
	s.restarts++
	delay := time.Duration(s.restarts) * 2 * time.Second
	s.state, s.err = lspFailed, "正在重启…"
	a.toast = s.spec.Name + " 已停止 · 正在重启(" + itoa(s.restarts) + "/3)"
	a.outAppend(outLSP, fmt.Sprintf("%s 退出,%.0f 秒后第 %d 次自动重启", s.spec.Name, delay.Seconds(), s.restarts))
	log.Info("语言服务器将自动重启", "server", s.spec.Name, "attempt", s.restarts, "delay", delay)
	time.AfterFunc(delay, func() {
		a.onUI(func() {
			if hub.sess[key] != s {
				return
			}
			lspStop(s.c, s.cmd) // 先取值再置 nil:goroutine 里不再回头读 s.c / s.cmd
			s.c, s.cmd = nil, nil
			s.state = lspStarting
			a.lspStart(key, s)
		})
	})
}
