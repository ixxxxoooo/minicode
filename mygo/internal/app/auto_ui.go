package app

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/automation"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

type autoLive struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	buf    strings.Builder
	at     time.Time
}

func (lv *autoLive) appendChunk(s string) {
	if s == "" {
		return
	}
	lv.mu.Lock()
	lv.buf.WriteString(s)
	lv.mu.Unlock()
}

func (lv *autoLive) snapshot() string {
	lv.mu.Lock()
	defer lv.mu.Unlock()
	return lv.buf.String()
}

// openAutoNewDlg 打开「新建自动化」模态(设计稿 behave.js au-new)。
func (a *App) openAutoNewDlg() {
	s := &a.au
	s.newDlg = true
	s.newErr, s.newErrAt, s.newErrVal = "", "", ""
	log.Info("打开新建自动化对话框")
	s.newName, s.newPrompt, s.newTime = "新的自动化", "", "09:00"
	s.newDays, s.newPerm = "每天", autoPerms[0]
	if n := a.currentProjectName(); n != "" {
		s.newProj = n
	}
}

func (a *App) closeAutoNewDlg() {
	log.Info("关闭新建自动化对话框")
	a.au.newDlg = false
}

func (a *App) submitAutoNew() {
	s := &a.au
	if strings.TrimSpace(s.newName) == "" {
		s.newErr, s.newErrAt, s.newErrVal = "请填写名称", "name", s.newName
		return
	}
	if strings.TrimSpace(s.newPrompt) == "" {
		s.newErr, s.newErrAt, s.newErrVal = "请写下要做什么", "prompt", s.newPrompt
		return
	}
	if !automation.ValidTime(s.newTime) {
		s.newErr, s.newErrAt, s.newErrVal = "时间格式如 09:30", "time", s.newTime
		return
	}
	n := a.auto.Add(s.newProj)
	n.Name, n.Prompt, n.Time, n.Days, n.Perm, n.Project = strings.TrimSpace(s.newName), strings.TrimSpace(s.newPrompt), s.newTime, s.newDays, s.newPerm, s.newProj
	if err := a.auto.Update(n); err != nil {
		s.newErr, s.newErrAt, s.newErrVal = err.Error(), "name", s.newName
		return
	}
	s.sel, s.draftID, s.focusName = n.ID, 0, false
	a.closeAutoNewDlg()
	a.toast = "已创建「" + n.Name + "」"
	log.Info("新建自动化", "id", n.ID)
}

// autoModals 自动化页浮层(新建 / 查看运行)。
func (a *App) autoModals(c *ui.Context) {
	a.autoNewDlg(c)
	a.autoRunView(c)
}

func (a *App) autoDlg(c *ui.Context, key string, w float32, title string, close func(), body func()) {
	if a.dlgClosing == key && time.Since(a.dlgCloseAt) >= kit.DurFast {
		a.dlgClosing, a.dlgSeen = "", ""
		close()
		return
	}
	want := (key == "auto-new" && a.au.newDlg) || (key == "au-run" && a.au.runView != "")
	t := a.tk
	a.dialogOpen(key, want, func(open *bool) {
		ui.DialogBase(c, open, func(backdrop, panel ui.Element) {
			backdrop.Background(t.Scrim).Justify(ui.Center).AlignItems(ui.Center)
			panel.Width(w).Background(t.Raised).Radius(t.RXl()).Column().Border(1, t.Bd1).Padding(14, 16).Gap(8)
			for _, sh := range t.Shadow3() {
				panel.Shadow(sh.X, sh.Y, sh.Blur, sh.Spread, sh.Color)
			}
			a.dialogIn(key, backdrop, panel)
			panel.Children(func() {
				ui.Row(c).AlignItems(ui.Center).Children(func() {
					ui.Text(c, title).FontSize(theme.FsHead).FontWeight(700).Grow(1)
					kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "关闭 Esc"}).OnClick(close)
				})
				body()
			})
		})
	})
}

// autoNewDlg 渲染新建模态。
func (a *App) autoNewDlg(c *ui.Context) {
	if !a.au.newDlg {
		return
	}
	a.autoDlg(c, "auto-new", 420, "新建自动化", a.closeAutoNewDlg, func() {
		t := a.tk
		s := &a.au
		lbl := func(text string) { ui.Text(c, text).FontSize(theme.FsCap).FontWeight(500).TextColor(t.Tx3) }
		// 错误只在对应字段还没被改动时显示:用户一动手修改就收起,不再残留「请写下要做什么」
		errOf := func(at string) string {
			if s.newErrAt != at {
				return ""
			}
			cur := map[string]string{"name": s.newName, "prompt": s.newPrompt, "time": s.newTime}[at]
			if cur != s.newErrVal {
				return ""
			}
			return s.newErr
		}
		kit.Field(c, "aun-name", kit.FieldOpt{Label: "名称", Err: errOf("name")}, func(box ui.Element) {
			box.Height(30)
			ui.TextInputBase(c.Key("aun-name"), &s.newName).Grow(1).FontSize(theme.FsBody)
		})
		ui.Column(c).Gap(4).Children(func() {
			lbl("提示词")
			pe := errOf("prompt")
			ta := ui.TextArea(c.Key("aun-prompt"), &s.newPrompt).Height(72)
			if pe != "" {
				ta.Border(1.5, t.Err) // 与 kit.Field 的错误态一致:红色描边
			}
			if e := pe; e != "" {
				ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
					kit.Ic(c, "alert", kit.S12).TextColor(t.Err)
					ui.Text(c, e).FontSize(10.5).TextColor(t.Err)
				})
			}
		})
		kinds := []string{"每天", "工作日", "每周一", "Cron"}
		cur := 0
		switch s.newDays {
		case "工作日":
			cur = 1
		case "周一":
			cur = 2
		case "0 9 * * 1", "0 9 * * *":
			if automation.KindOf(s.newDays) == automation.KCron {
				cur = 3
			}
		}
		if i := kit.Seg(c, []kit.SegItem{{Text: kinds[0]}, {Text: kinds[1]}, {Text: kinds[2]}, {Text: kinds[3]}}, cur, kit.SegOpt{Key: "aun-kind", Fill: true}); i >= 0 {
			s.newDays = []string{"每天", "工作日", "周一", "0 9 * * *"}[i]
		}
		kit.Field(c, "aun-time", kit.FieldOpt{Label: "时间", Err: errOf("time")}, func(box ui.Element) {
			box.Height(30).Width(120).Grow(0).Shrink(0)
			ui.TextInputBase(c.Key("aun-time"), &s.newTime).Grow(1).FontSize(theme.FsBody)
		})
		ui.Column(c).Gap(4).Children(func() {
			lbl("项目")
			a.autoProjPick(c, &s.newProj, "aun-proj")
		})
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			kit.Btn(c, "取消", kit.BtnOpt{Variant: kit.BtnGhost}).OnClick(a.closeAutoNewDlg)
			kit.Btn(c, "创建", kit.BtnOpt{Variant: kit.BtnPri}).OnClick(a.submitAutoNew)
		})
	})
}

// autoProjPick 项目下拉(列出侧栏已知项目)。
func (a *App) autoProjPick(c *ui.Context, sel *string, key string) {
	t := a.tk
	names := a.projectNames()
	if len(names) == 0 {
		*sel = ""
		ui.Text(c, "还没有项目,请先添加").FontSize(theme.FsCap).TextColor(t.Tx4)
		return
	}
	if *sel == "" {
		*sel = names[0]
	}
	idx := 0
	for i, n := range names {
		if n == *sel {
			idx = i
			break
		}
	}
	// 项目很多时分段控件会把名字挤成「ro…」,改成可折行的选择 chip。
	// 项目一多就限高滚动,别把抽屉底部的「立即运行 / 保存」顶出去。
	ui.Scroll(c).Key(key + "-scroll").MaxHeight(84).Children(func() {
		ui.Row(c).Gap(5).Wrap().Children(func() {
			for i, n := range names {
				i, n := i, n
				on := i == idx
				b := ui.ButtonBase(c).Key(key + "-" + itoa(i)).Height(24).PaddingX(9).Radius(12).AlignItems(ui.Center).
					Cursor(ui.CursorPointer).FontSize(theme.FsCap).Transition(kit.Fade)
				switch {
				case on:
					b.Background(t.AcSoft).TextColor(t.AcTx).Border(1, t.Ac)
				case b.Hovered():
					b.Background(t.Fill2).TextColor(t.Tx2).Border(1, t.Bd2)
				default:
					b.Background(ui.Transparent).TextColor(t.Tx3).Border(1, t.Bd2)
				}
				b.OnClick(func() { *sel = n })
				b.Children(func() { ui.Text(c, n).SingleLine() })
			}
		})
	})
}

func (a *App) projectNames() []string {
	var out []string
	seen := map[string]bool{}
	for _, p := range a.projects {
		if p.Name != "" && !seen[p.Name] {
			seen[p.Name] = true
			out = append(out, p.Name)
		}
	}
	return out
}

// autoPermPick 权限下拉。
func (a *App) autoPermPick(c *ui.Context, perm *string) {
	idx := 0
	for i, x := range autoPerms {
		if x == *perm {
			idx = i
			break
		}
	}
	items := make([]kit.SegItem, len(autoPerms))
	for i, p := range autoPerms {
		items[i] = kit.SegItem{Text: p}
	}
	if i := kit.Seg(c, items, idx, kit.SegOpt{Key: "au-perm", Fill: true}); i >= 0 {
		*perm = autoPerms[i]
	}
}

// autoRunView 运行记录只读查看(点最近运行进入)。
func (a *App) autoRunView(c *ui.Context) {
	s := &a.au
	if s.runView == "" {
		return
	}
	id, _ := parseRunView(s.runView)
	it, ok := a.auto.Get(id)
	title := "运行输出"
	if ok {
		title = it.Name + " · 运行记录"
	}
	a.autoDlg(c, "au-run", 520, title, func() { s.runView = "" }, func() {
		t := a.tk
		var body string
		var live bool
		var rec automation.Run
		if lv := a.autoLive[id]; lv != nil && s.runView == liveKey(id) {
			live = true
			body = lv.snapshot()
		}
		if ok {
			for _, r := range it.Runs {
				if r.Key == s.runView || (r.Key == "" && fmt.Sprintf("auto:%d:%d", id, r.At.Unix()) == s.runView) {
					rec = r
					body, _ = automation.SplitSession(r.Output)
					if r.Err != "" && body == "" {
						body = r.Err
					}
					break
				}
			}
		}
		if live {
			ui.Column(c).Gap(6).Children(func() {
				kit.IndetMeter(c)
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					kit.Spin(c, t.Ac)
					ui.Text(c, "运行中…").TextColor(t.Ac)
					ui.Spacer(c).Grow(1)
					kit.Btn(c, "取消运行", kit.BtnOpt{Small: true, Variant: kit.BtnDangerGhost}).OnClick(func() { a.cancelAutoRun(id) })
				})
			})
		} else if sid := autoRunSessionID(rec); sid != "" {
			kit.Btn(c, "查看会话", kit.BtnOpt{Variant: kit.BtnPri, Icon: "chat"}).OnClick(func() {
				a.openAutomationSession(sid)
				s.runView = ""
			})
		} else if why := autoNoSessionReason(rec); why != "" {
			ui.Text(c, why).FontSize(theme.FsCap).TextColor(t.Tx4)
		}
		box := ui.Column(c).Background(t.Sunken).Radius(t.RSm()).Padding(10, 12).MaxHeight(360)
		box.Children(func() {
			if body == "" {
				ui.Text(c, map[bool]string{true: "正在运行,输出会出现在这里…", false: "没有输出"}[live]).TextColor(t.Tx3).LineHeight(1.55)
			} else {
				ui.Text(c, body).FontFeatures("mono").FontSize(theme.FsCap).LineHeight(1.55).TextColor(t.Tx2)
			}
		})
	})
}

func liveKey(id int) string { return fmt.Sprintf("auto:%d:live", id) }

func parseRunView(k string) (id int, rest string) {
	if strings.HasPrefix(k, "auto:") {
		parts := strings.SplitN(k, ":", 3)
		if len(parts) >= 2 {
			id = atoi(parts[1])
		}
	}
	return id, k
}

func (a *App) cancelAutoRun(id int) {
	if lv := a.autoLive[id]; lv != nil && lv.cancel != nil {
		lv.cancel()
	}
}
