package app

import (
	"context"
	"fmt"
	"time"

	"github.com/ixxxxoooo/minicode/internal/automation"
	"github.com/ixxxxoooo/minicode/internal/claude"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/diag"
)

// runAutomationNow 自动化页「立即运行」(带会话 id 记录);定时任务仍走 app.runAutomation。
func (a *App) runAutomationNow(it automation.Automation) {
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
			sessionID = demo.AutomationDemoSessionID
		default:
			end := diag.SpanSlow("app", "运行自动化(stream)", 30*time.Second, "name", it.Name)
			res, e := claude.RunAutomationPromptStreamCtx(ctx, dir, it.Prompt, claude.PermModeOf(it.Perm), 15*time.Minute, lv.appendChunk)
			end()
			out, sessionID, err = res.Output, res.SessionID, e
			if err != nil && runCtxFn != nil && sessionID == "" && out == "" {
				// 无 stream-json 能力时回落注入的运行器(无会话 id)
				out, err = runCtxFn(ctx, it, dir, lv.appendChunk)
				out, _ = automation.SplitSession(out)
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

// openAutoRun 打开一条运行记录:有会话 id 则进会话,否则只读输出浮层。
func (a *App) openAutoRun(r automation.Run, rk string) {
	if r.SessionID != "" {
		a.OpenSession(r.SessionID)
		return
	}
	_, sid := automation.SplitSession(r.Output)
	if sid != "" {
		a.OpenSession(sid)
		return
	}
	a.au.runView = rk
}
