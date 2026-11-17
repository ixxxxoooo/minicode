package app

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// sessLoadState 记录「正在打开已有会话」:回放完成前不显示欢迎页,并可选暂留上一会话画面。
type sessLoadState struct {
	active bool
	hold   *core.State // 切换前会话的条目(有内容时短暂继续展示)
	err    string      // 历史读取失败
	to     string
	at     time.Time
}

func openSessionIsNew(id string) bool {
	return id == "" || strings.HasPrefix(id, "new:")
}

// showChatWelcome 是否应显示新会话欢迎页(仅无 id 且无在途载入)。
func (a *App) showChatWelcome() bool {
	if a.st == nil || len(a.st.Items) > 0 {
		return false
	}
	if a.sessLoad.active || a.sessLoad.err != "" {
		return false
	}
	return a.curSess == ""
}

func (a *App) beginSessLoad(id string) {
	if openSessionIsNew(id) {
		a.sessLoad = sessLoadState{}
		return
	}
	hold := (*core.State)(nil)
	if a.st != nil && len(a.st.Items) > 0 {
		hold = a.st
	}
	a.sessLoad = sessLoadState{active: true, hold: hold, to: normalizeSessID(id), at: time.Now()}
}

func normalizeSessID(id string) string {
	if strings.HasPrefix(id, "fork:") {
		if i := strings.LastIndex(id, ":"); i > len("fork:") {
			return id[i+1:]
		}
	}
	return id
}

func (a *App) finishSessLoad() {
	if !a.sessLoad.active {
		return
	}
	elapsed := time.Since(a.sessLoad.at)
	log.Info("会话载入完成", "to", a.curSess, "items", len(a.st.Items), "耗时", elapsed)
	a.sessLoad.hold = nil
	a.sessLoad.active = false
	a.triggerFlowRise()
}

func (a *App) onSessLoadEvent(ev core.Event) {
	if n, ok := ev.(core.EvNotice); ok && a.sessLoad.active && strings.Contains(n.Text, "读取历史") {
		elapsed := time.Since(a.sessLoad.at)
		a.sessLoad.err = n.Text
		a.sessLoad.active = false
		a.sessLoad.hold = nil
		log.Warn("会话载入失败", "to", a.sessLoad.to, "耗时", elapsed, "err", n.Text)
		return
	}
	if _, ok := ev.(core.EvReady); ok && a.sessLoad.active {
		a.finishSessLoad()
	}
}

// flowDisplayState 载入期间有旧内容则暂用旧条目画会话流(输入框等仍绑当前 a.st)。
func (a *App) flowDisplayState() *core.State {
	if a.st != nil && len(a.st.Items) > 0 {
		return a.st
	}
	if a.sessLoad.active && a.sessLoad.hold != nil && len(a.sessLoad.hold.Items) > 0 {
		return a.sessLoad.hold
	}
	return a.st
}

func (a *App) chatHeadTitle() string {
	if a.st != nil && a.st.Title != "" {
		return a.st.Title
	}
	if a.curSess != "" {
		return a.sessionTitle(a.curSess)
	}
	return "新对话"
}

// chatSessionLoading 打开已有会话、回放尚未到达时的占位(骨架 + 转圈,非欢迎页)。
func (a *App) chatSessionLoading(c *ui.Context) {
	t := a.tk
	a.chatHead(c, a.chatHeadTitle())
	kit.VScroll(c).Key("flow").TrackScroll(&a.flowSt).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Grow(1).Padding(16, 20).Gap(12).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				kit.SpinStyled(c, t.Tx3)
				ui.Text(c, "正在载入会话…").TextColor(t.Tx3)
			})
			kit.SkelRows(c, 5)
		})
	})
	a.chatDockColumn(c)
}

func (a *App) chatSessionLoadFailed(c *ui.Context) {
	t := a.tk
	msg := a.sessLoad.err
	if msg == "" {
		msg = "无法打开此会话"
	}
	a.chatHead(c, a.chatHeadTitle())
	kit.VScroll(c).Key("flow").TrackScroll(&a.flowSt).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Grow(1).Padding(20, 24).Gap(8).Justify(ui.Center).Children(func() {
			kit.Ic(c, "alert", kit.S12).TextColor(t.Warn)
			ui.Text(c, msg).TextColor(t.Tx2).Wrap()
			kit.Btn(c, "返回侧栏另选会话", kit.BtnOpt{Variant: kit.BtnGhost}).OnClick(func() {
				a.sessLoad.err = ""
				a.NewSession()
			})
		})
	})
	a.chatDockColumn(c)
}

// chatEmptyOpened 已有会话 id、回放后仍无条目:只保留标题行与输入框,不回落欢迎页。
func (a *App) chatEmptyOpened(c *ui.Context) {
	a.chatHead(c, a.chatHeadTitle())
	ui.Box(c).Grow(1).MinHeight(0)
	a.chatDockColumn(c)
}

func (a *App) chatDockColumn(c *ui.Context) {
	dockBox := ui.Column(c).Key("chat-dock").Padding(4, 0, 10, 0).Shrink(0).AlignItems(ui.Center)
	a.dockH = dockBox.Bounds().H
	dockBox.Children(func() {
		a.flowCol(c, func() {
			a.syncDockComposer()
			a.taskDock(c)
			a.todoDock(c)
			if !a.dockCard(c) {
				a.queueCard(c)
				a.composer(c)
			}
		})
	})
}

// withFlowDisplay 仅在会话流绘制期间临时切换用于展示的 State。
func (a *App) withFlowDisplay(fn func()) {
	disp := a.flowDisplayState()
	if disp == nil || disp == a.st {
		fn()
		return
	}
	prev := a.st
	a.st = disp
	defer func() { a.st = prev }()
	fn()
}
