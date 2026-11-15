package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/claude"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 回合操作条:复制 / 重试 / 分叉。

func (a *App) turnText(uid string) string {
	var buf strings.Builder
	for i, it := range a.st.Items {
		if u, ok := it.(*core.User); ok && u.ID == uid {
			for _, x := range a.st.Items[i+1:] {
				if _, ok := x.(*core.User); ok {
					break
				}
				if as, ok := x.(*core.Assistant); ok {
					buf.WriteString(as.Text)
				}
			}
			break
		}
	}
	return strings.TrimSpace(buf.String())
}

func (a *App) userText(uid string) string {
	for _, it := range a.st.Items {
		if u, ok := it.(*core.User); ok && u.ID == uid {
			return u.Text
		}
	}
	return ""
}

func (a *App) turnFooterActions(c *ui.Context, uid string) {
	t := a.tk
	stats, ok := a.st.TurnStats[uid]
	if !ok {
		return
	}
	showStats := a.prefs.Bool(pStats, true)
	canFork := a.curSess != "" && !strings.HasPrefix(a.curSess, "new:")
	ui.Row(c).Key("foot-"+uid).Height(22).Margin(0, 0, 0, -4).Gap(2).AlignItems(ui.Center).TextColor(t.Tx4).FontSize(theme.FsCap).Children(func() {
		if kit.Ib(c, "copy", kit.IbOpt{Small: true, Tip: "复制本回合回复"}).Clicked() {
			a.copyOut(a.turnText(uid), "本回合回复")
		}
		if a.st.Phase != core.PhaseRunning && kit.Ib(c, "refresh", kit.IbOpt{Small: true, Tip: "重试:重新发送该回合的提问"}).Clicked() {
			if txt := a.userText(uid); txt != "" {
				a.send(core.CmdPrompt{Text: txt})
				log.Info("重试回合", "user", uid)
			}
		} else if a.st.Phase == core.PhaseRunning {
			kit.Ib(c, "refresh", kit.IbOpt{Small: true, Tip: "回合进行中,结束后才能重试"}).Opacity(0.35)
		}
		fb := kit.Ib(c, "fork", kit.IbOpt{Small: true, Tip: map[bool]string{true: "从这里分叉为新会话", false: "需要有已保存的会话 id"}[canFork]})
		if canFork {
			fb.OnClick(func() { a.forkTurn(uid) })
		} else {
			fb.Opacity(0.35).Cursor(ui.CursorDefault)
		}
		if !showStats {
			return
		}
		var parts []string
		if l := stats.ModelLabel(); l != "" {
			parts = append(parts, l)
		}
		if d, ok := a.st.Durations[uid]; ok && d > 0 {
			parts = append(parts, core.FormatElapsed(int64(d)*1000))
		}
		parts = append(parts, strings.Split(stats.ShortText(), " · ")...)
		ui.Row(c).Margin(0, 0, 0, 6).Gap(6).AlignItems(ui.Center).Children(func() {
			for i, p := range parts {
				if i > 0 {
					ui.Text(c, "·")
				}
				ui.Text(c, p).FontFeatures("tnum").SingleLine()
			}
		})
	})
}

func (a *App) forkTurn(atUser string) {
	parent := a.curSess
	if parent == "" {
		a.toast = "新会话还不能分叉"
		return
	}
	newID := claude.NewSessionID()
	if a.be != nil && a.be.Name() == "demo" {
		a.forkTurnDemo(parent, newID, atUser)
		return
	}
	a.send(core.CmdForkSession{ParentID: parent, NewID: newID, AtUser: atUser})
	a.OpenSession(fmt.Sprintf("fork:%s:%s", parent, newID))
	a.toast = "已分叉为新会话"
	log.Info("分叉会话", "parent", parent, "new", newID, "at", atUser)
}

func (a *App) forkTurnDemo(parent, newID, atUser string) {
	a.OpenSession("fork:" + parent + ":" + newID)
	if txt, ok := a.st.TruncateBeforeUser(atUser); ok {
		_ = txt
	}
	a.toast = "已分叉(演示)"
}
