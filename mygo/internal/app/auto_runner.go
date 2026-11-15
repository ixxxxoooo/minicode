package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/automation"
	"github.com/ixxxxoooo/minicode/internal/claude"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/diag"
)

// runAutomationCore 立即运行与定时到点共用:流式运行、回写 session_id 并落盘。
func (a *App) runAutomationCore(it automation.Automation) {
	log := diag.L("app")
	if a.autoRun == nil && a.autoRunCtx == nil && !a.demoWs() {
		a.auto.RunNow(it.ID, a.now())
		a.toast = "已开始运行「" + it.Name + "」"
		return
	}
	if a.autoBusy == nil {
		a.autoBusy = map[int]bool{}
	}
	if a.autoBusy[it.ID] {
		a.toast = "「" + it.Name + "」正在运行"
		return
	}
	dir := a.projectDir(it.Project)
	if dir == "" {
		a.auto.Record(it.ID, automation.Run{At: a.now(), Err: "找不到项目「" + it.Project + "」的目录"})
		a.toast = "运行失败:找不到项目目录"
		return
	}
	if a.autoLive == nil {
		a.autoLive = map[int]*autoLive{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	lv := &autoLive{cancel: cancel, at: a.now()}
	a.autoLive[it.ID] = lv
	a.autoBusy[it.ID] = true
	a.au.runView = liveKey(it.ID)
	a.toast = "已开始运行「" + it.Name + "」"
	runCtxFn := a.autoRunCtx
	a.bg("automation", func() func() {
		var out string
		var sessionID string
		var err error
		switch {
		case a.demoWs():
			out, err = demo.RunAutomationDemo(ctx, it, dir, lv.appendChunk)
			var sid string
			out, sid = automation.SplitSession(out)
			sessionID = sid
			if sessionID == "" {
				sessionID = demo.AutomationDemoSessionID
			}
		default:
			_, hasClaude := claude.FindClaude("")
			switch {
			case !hasClaude && runCtxFn != nil:
				out, err = runCtxFn(ctx, it, dir, lv.appendChunk)
				out, sessionID = automation.SplitSession(out)
			case !hasClaude && a.autoRun != nil:
				out, err = runCtxLegacy(ctx, it, dir, a.autoRun, lv)
				out, sessionID = automation.SplitSession(out)
			default:
				end := diag.SpanSlow("app", "运行自动化(stream)", 30*time.Second, "name", it.Name)
				res, e := claude.RunAutomationPromptStreamCtx(ctx, dir, it.Prompt, claude.PermModeOf(it.Perm), 15*time.Minute, lv.appendChunk)
				end()
				out, sessionID, err = res.Output, res.SessionID, e
				if sessionID == "" && out != "" {
					out, sessionID = automation.SplitSession(out)
				}
				if err != nil && sessionID == "" && out == "" {
					if runCtxFn != nil {
						out, err = runCtxFn(ctx, it, dir, lv.appendChunk)
						out, sessionID = automation.SplitSession(out)
					} else if a.autoRun != nil {
						out, err = runCtxLegacy(ctx, it, dir, a.autoRun, lv)
						out, sessionID = automation.SplitSession(out)
					}
				}
			}
		}
		r := automation.Run{At: time.Now(), OK: err == nil, SessionID: sessionID}
		if err != nil {
			r.Err = err.Error()
			if ctx.Err() == context.Canceled {
				r.Err = "已取消"
			}
		}
		if out == "" {
			out = lv.snapshot()
		}
		out, _ = automation.SplitSession(out)
		if rs := []rune(out); len(rs) > 8000 {
			out = string(rs[len(rs)-8000:])
		}
		r.Output = out
		r.Key = fmt.Sprintf("auto:%d:%d", it.ID, r.At.Unix())
		return func() {
			delete(a.autoBusy, it.ID)
			delete(a.autoLive, it.ID)
			a.auto.Record(it.ID, r)
			if r.OK {
				a.toast = "「" + it.Name + "」运行完成"
				if sessionID != "" {
					a.toast += " · 可查看会话"
				}
			} else {
				a.toast = "「" + it.Name + "」运行失败:" + r.Err
			}
			log.Info("自动化运行结束", "id", it.ID, "session", sessionID, "ok", r.OK)
		}
	})
}

// runAutomationNow 自动化页「立即运行」按钮入口。
func (a *App) runAutomationNow(it automation.Automation) { a.runAutomationCore(it) }

func autoRunSessionID(r automation.Run) string {
	if r.SessionID != "" {
		return r.SessionID
	}
	_, sid := automation.SplitSession(r.Output)
	return sid
}

func autoNoSessionReason(r automation.Run) string {
	if autoRunSessionID(r) != "" {
		return ""
	}
	switch {
	case r.Err == "已取消":
		return "已取消,未产生会话"
	case !r.OK:
		return "运行失败,未产生会话"
	default:
		return "没有关联会话(旧记录或未启动成功)"
	}
}

// openAutomationSession 打开自动化关联的会话;索引未就绪时触发一次重扫。
func (a *App) openAutomationSession(id string) {
	id = strings.TrimSpace(id)
	if id == "" {
		a.toast = "此次运行没有关联会话"
		return
	}
	if _, _, ok := a.sessionByID(id); ok {
		a.OpenSessionChecked(id)
		return
	}
	if a.demoWs() {
		a.OpenSessionChecked(id)
		return
	}
	if a.projRefresh == nil {
		a.toast = "会话正在索引,请稍后再试「查看会话」"
		log.Warn("自动化会话未在索引中", "session", id)
		return
	}
	a.SetProjScanning(true)
	a.bg("会话索引", func() func() {
		ps := a.projRefresh()
		return func() {
			a.SetProjScanning(false)
			a.RefreshProjects(ps)
			if _, _, ok := a.sessionByID(id); ok {
				a.OpenSessionChecked(id)
				log.Info("自动化打开会话", "session", id)
			} else {
				a.toast = "会话正在索引,请稍后再试「查看会话」"
				log.Warn("自动化会话未在索引中", "session", id)
			}
		}
	})
}

// openAutoRun 打开一条运行记录:有会话 id 则进会话,否则只读输出浮层。
func (a *App) openAutoRun(r automation.Run, rk string) {
	if sid := autoRunSessionID(r); sid != "" {
		a.openAutomationSession(sid)
		return
	}
	a.au.runView = rk
}
