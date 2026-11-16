package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diff"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 待处理卡片(设计稿 .dockcard):权限 / 提问就地替换输入框,处理完恢复输入框;
// 计划卡留在会话流里(.plan)。

// qaState 是提问向导的本地进度:当前第几题、每题选中的下标与自由文本。
type qaState struct {
	cur   int
	sel   [][]int
	freet []string // 每题「其它」或纯自由输入
	skip  []bool   // 每题是否被「跳过」
}

// dockCard 画待处理卡;返回是否画了(画了就不再画输入框)。
func (a *App) dockCard(c *ui.Context) bool {
	switch v := a.pendingDock().(type) {
	case *core.PermissionRequest:
		a.permCard(c, v)
		return true
	case *core.QuestionRequest:
		a.askCard(c, v)
		return true
	}
	return false
}

// cardShell 是卡片外壳:raised 底、强调色细描边、阴影 2。
func (a *App) cardShell(c *ui.Context, key string) ui.Element {
	t := a.tk
	b := ui.Column(c).Key(key).Background(t.Raised).Radius(t.RLg()).Border(1, t.AcLine).Padding(10, 12).Gap(8)
	for _, s := range t.Shadow2() {
		b.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	return b
}

// permTitle 按工具种类给出问句。
func permTitle(r *core.PermissionRequest) string {
	if r.Tool == nil {
		return "允许这次操作吗?"
	}
	switch r.Tool.Kind {
	case core.KindExecute:
		return "允许运行这条命令吗?"
	case core.KindEdit, core.KindWrite:
		return "允许修改这个文件吗?"
	case core.KindRead:
		return "允许读取这个文件吗?"
	case core.KindFetch:
		return "允许访问这个网址吗?"
	}
	return "允许使用 " + r.Tool.Name + " 吗?"
}

// permToolTag 是权限卡右上角的工具标签:内置工具写中文,MCP / 其他工具保留原名。
func permToolTag(t *core.ToolCall) string {
	switch t.Name {
	case "Edit", "MultiEdit", "NotebookEdit":
		return "编辑"
	case "Write":
		return "写入"
	case "Read":
		return "读取"
	case "Bash":
		return "命令"
	case "WebFetch":
		return "网页"
	case "WebSearch":
		return "搜索网页"
	case "Glob", "Grep":
		return "搜索"
	}
	return t.Name
}

// permDetail 是卡片里的等宽详情:命令 / 路径 / 网址。
func (a *App) permDetail(r *core.PermissionRequest) string {
	if r.Tool == nil {
		return ""
	}
	for _, k := range []string{"command", "file_path", "url", "pattern"} {
		if s := inputStr(r.Tool, k); s != "" {
			if k == "file_path" {
				return a.shortPath(s)
			}
			return s
		}
	}
	return r.Tool.Title
}

func (a *App) permCard(c *ui.Context, r *core.PermissionRequest) {
	t := a.tk
	ok, deny := "", ""
	for _, o := range r.Options {
		if o.Kind == core.AllowOnce && ok == "" {
			ok = o.ID
		}
		if o.Kind.Rejects() && deny == "" {
			deny = o.ID
		}
	}
	resolve := func(id string) {
		if id == "" {
			return
		}
		a.send(core.CmdResolvePermission{ID: r.ID, Option: id})
	}
	if !a.paletteOn && a.screen == ScreenWork {
		if c.Shortcut(0, ui.KeyEnter) {
			resolve(ok)
		} else if c.Shortcut(0, ui.KeyEscape) {
			resolve(deny)
		}
	}
	a.cardShell(c, "dock-perm").Children(func() {
		ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
			kit.Ic(c, "shield", kit.S12).TextColor(t.AcTx)
			ui.Text(c, permTitle(r)).FontWeight(700).SingleLine().Shrink(1)
			ui.Spacer(c).Grow(1)
			a.dockBadge(c)
			if r.Tool != nil {
				kit.Tag(c, permToolTag(r.Tool), kit.KAc)
			}
		})
		if d := a.permDetail(r); d != "" {
			filePath := ""
			if r.Tool != nil {
				filePath = inputStr(r.Tool, "file_path")
			}
			if filePath != "" {
				ui.Row(c).Gap(6).Padding(6, 8).AlignItems(ui.Center).Background(t.Sunken).Radius(t.RSm()).Border(1, t.Bd1).Children(func() {
					a.fileIc(c, baseName(filePath))
					ui.Text(c, d).Font("monospace").FontSize(11.5).LineHeight(1.5).TextColor(t.Tx1).Grow(1).SingleLine()
				})
			} else {
				ui.Column(c).Background(t.Sunken).Radius(t.RSm()).Border(1, t.Bd1).Padding(6, 9).Children(func() {
					ui.Text(c, d).Font("monospace").FontSize(11.5).LineHeight(1.5).TextColor(t.Tx1)
				})
			}
		}
		a.permDiffPreview(c, r)
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			for _, o := range r.Options {
				v := kit.BtnDefault
				switch {
				case o.ID == ok:
					v = kit.BtnPri
				case o.Kind.Rejects():
					v = kit.BtnGhost
				}
				id := o.ID
				kit.Btn(c, o.Label, kit.BtnOpt{Variant: v, Small: true}).OnClick(func() { resolve(id) })
			}
			ui.Spacer(c).Grow(1)
			if a.mode == ModeAgent {
				ui.Row(c).Gap(4).AlignItems(ui.Center).TextColor(t.Tx3).FontSize(theme.FsCap).Children(func() {
					kit.Kbd(c, "⏎")
					ui.Text(c, "允许")
					kit.Kbd(c, "esc")
					ui.Text(c, "拒绝")
				})
			}
		})
	})
}

// ───────── 提问向导 ─────────

func (a *App) qaOf(q *core.QuestionRequest) *qaState {
	if a.qa == nil {
		a.qa = map[string]*qaState{}
	}
	s := a.qa[q.ID]
	if s == nil {
		s = &qaState{sel: make([][]int, len(q.Questions)), freet: make([]string, len(q.Questions)), skip: make([]bool, len(q.Questions))}
		a.qa[q.ID] = s
	}
	return s
}

func isOtherLabel(label string) bool {
	label = strings.TrimSpace(label)
	return label == "其他" || label == "其它" || strings.HasPrefix(label, "其它")
}

func (s *qaState) wantsFreeInput(qu core.Question) bool { return s.wantsFreeAt(qu, s.cur) }

// wantsFreeAt 第 i 题是否需要自由输入(无选项,或选了「其它」)。
func (s *qaState) wantsFreeAt(qu core.Question, i int) bool {
	if len(qu.Options) == 0 {
		return true
	}
	for _, k := range s.sel[i] {
		if k >= 0 && k < len(qu.Options) && isOtherLabel(qu.Options[k].Label) {
			return true
		}
	}
	return false
}

func hasInt(l []int, v int) bool {
	for _, x := range l {
		if x == v {
			return true
		}
	}
	return false
}

// toggleOption 单选替换、多选切换。
func (s *qaState) toggleOption(q core.Question, i int) {
	cur := s.sel[s.cur]
	if q.MultiSelect {
		for k, x := range cur {
			if x == i {
				s.sel[s.cur] = append(cur[:k:k], cur[k+1:]...)
				return
			}
		}
		s.sel[s.cur] = append(cur, i)
		return
	}
	s.sel[s.cur] = []int{i}
}

// answers 把选中的下标换成选项文本;自由输入题或「其它」附带说明。
func (s *qaState) answers(q *core.QuestionRequest) [][]string {
	out := make([][]string, len(q.Questions))
	for i, qu := range q.Questions {
		if i < len(s.skip) && s.skip[i] {
			continue
		}
		if len(qu.Options) == 0 {
			if t := strings.TrimSpace(s.freet[i]); t != "" {
				out[i] = []string{t}
			}
			continue
		}
		other := false
		for _, k := range s.sel[i] {
			if k < len(qu.Options) {
				lb := qu.Options[k].Label
				if isOtherLabel(lb) {
					other = true
					continue
				}
				out[i] = append(out[i], lb)
			}
		}
		if other {
			if t := strings.TrimSpace(s.freet[i]); t != "" {
				out[i] = append(out[i], t)
			} else {
				out[i] = append(out[i], "其他")
			}
		}
	}
	return out
}

// ───────── 计划卡(会话流里) ─────────

func planTitle(md string) string {
	for _, ln := range strings.Split(md, "\n") {
		ln = strings.TrimSpace(ln)
		if strings.HasPrefix(ln, "#") {
			return strings.TrimSpace(strings.TrimLeft(ln, "#"))
		}
	}
	return "实施计划"
}

// planCardBody 去掉计划开头的标题行(标题已在卡片头部显示),卡片里只留摘要。
func planCardBody(md string) string {
	lines := strings.Split(md, "\n")
	for i, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(ln), "#") {
			return strings.TrimLeft(strings.Join(lines[i+1:], "\n"), "\n")
		}
		break
	}
	return md
}

func (a *App) planCard(c *ui.Context, p *core.PlanApproval) {
	t := a.tk
	box := ui.Column(c).Key("plan-"+p.ID).Background(t.Raised).Radius(t.RMd()).Padding(8, 11).Gap(5)
	for _, s := range t.Shadow1() {
		box.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	box.Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			kit.Ic(c, "list", kit.S12).TextColor(t.Warn)
			ui.Text(c, planTitle(p.Markdown)).FontWeight(700).SingleLine().Shrink(1)
			switch {
			case p.Streaming:
				kit.Tag(c, "生成中…")
			case p.Approved == nil:
				kit.Tag(c, "待批准", kit.KWarn)
			case *p.Approved:
				kit.Tag(c, "已批准", kit.KOk)
			default:
				kit.Tag(c, "已拒绝")
			}
			ui.Spacer(c).Grow(1)
			kit.Btn(c, "查看", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "ext", Tip: "在工作区打开计划"}).OnClick(func() { a.OpenWorkspaceTab(vPlan, p.ID) })
		})
		if body := planCardBody(p.Markdown); strings.TrimSpace(body) != "" {
			if p.Streaming {
				markdown.Stream(c, body, a.mdOptions())
			} else {
				markdown.View(c, body, a.mdOptions())
			}
		}
		if !p.Streaming && p.Approved == nil {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				resolve := func(ok bool, m core.PermissionMode) {
					mm := m
					a.send(core.CmdResolvePlan{ID: p.ID, Approve: ok, Mode: &mm})
				}
				kit.Btn(c, "开始实施", kit.BtnOpt{Variant: kit.BtnPri, Small: true, Icon: "play"}).OnClick(func() { resolve(true, core.ModeDefault) })
				kit.Btn(c, "自动模式", kit.BtnOpt{Small: true}).OnClick(func() { resolve(true, core.ModeAuto) })
				kit.Btn(c, "继续规划", kit.BtnOpt{Variant: kit.BtnGhost, Small: true}).OnClick(func() { resolve(false, a.st.Mode) })
			})
		}
	})
}

// ───────── 待发送区(排队 + 引导) ─────────

// queueKind 是待发送行左侧的类型标签(设计稿 .qk):引导 = 强调淡底 + 闪电,排队 = 灰底 + 序号。
func (a *App) queueKind(c *ui.Context, steer bool, n int) {
	t := a.tk
	bg, fg := t.Fill2, t.Tx2
	if steer {
		bg, fg = t.AcSoft, t.AcTx
	}
	ui.Row(c).Height(17).PaddingX(6).Gap(4).Radius(9).Background(bg).AlignItems(ui.Center).Shrink(0).Children(func() {
		if steer {
			kit.Ic(c, "bolt", kit.S12).TextColor(fg)
			ui.Text(c, "引导").FontSize(11).TextColor(fg)
			return
		}
		ui.Text(c, itoa(n)).FontSize(11).TextColor(fg).FontFeatures("tnum")
		ui.Text(c, "排队").FontSize(11).TextColor(fg)
	})
}

// queueCard 是输入框正上方的「待发送区」(设计稿 flow-queue):排队与引导共用。
// 引导 = 已发给 Claude、等它接收(接收后才进会话流);排队 = 等本轮结束后依次发送。
// 排队行悬停有:改为引导 / 停止并发送 / 删除,点文字退回输入框编辑;暂停时顶部有「继续 / 撤回」。
func (a *App) queueCard(c *ui.Context) {
	nq, ns := a.q.len(), len(a.q.steers)
	if nq+ns == 0 {
		return
	}
	t := a.tk
	box := ui.Column(c).Key("queue").Margin(0, 12, 0, 12).Background(t.Raised).Padding(4).Gap(1).Radius(t.RMd(), t.RMd(), 0, 0)
	for _, s := range t.Shadow1() {
		box.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	box.Children(func() {
		ui.Row(c).Gap(8).Padding(2, 6, 3, 6).AlignItems(ui.Center).TextColor(t.Tx3).FontSize(theme.FsCap).Children(func() {
			kit.Ic(c, "clock", kit.S12)
			ui.Text(c, "待发送 "+itoa(nq+ns)).FontWeight(600).TextColor(t.Tx2)
			ui.Spacer(c).Grow(1)
			if a.chatTier() != tierNarrow {
				e, m := a.busyKeys()
				kit.Kbd(c, "⏎")
				ui.Text(c, e)
				kit.Kbd(c, "⌘⏎")
				ui.Text(c, m)
			}
		})
		if a.q.paused && nq > 0 {
			ui.Row(c).Gap(6).Padding(2, 6).AlignItems(ui.Center).Children(func() {
				kit.Ic(c, "alert", kit.S12).TextColor(t.Warn)
				ui.Text(c, "回合被中断,排队已暂停").FontSize(theme.FsCap).TextColor(t.Tx2).Shrink(1)
				ui.Spacer(c).Grow(1)
				kit.Btn(c, "继续", kit.BtnOpt{Variant: kit.BtnSoft, Small: true}).OnClick(func() { a.q.paused = false; a.drainQueue() })
				kit.Btn(c, "撤回", kit.BtnOpt{Variant: kit.BtnGhost, Small: true}).OnClick(func() { a.q.clear() })
			})
		}
		for k, sp := range a.q.steers { // 引导靠前:已经发给 Claude
			sp := sp
			steerImg := sp.Show == "(图片)"
			ui.Row(c).Key("steer" + itoa(k)).Height(26).PaddingX(6).Gap(8).AlignItems(ui.Center).Radius(t.RSm()).Children(func() {
				a.queueKind(c, true, 0)
				ui.Text(c, sp.Show).SingleLine().Grow(1).MinWidth(0)
				if steerImg {
					a.queueImgBadge(c, 1)
				}
				ui.Row(c).Gap(5).AlignItems(ui.Center).Shrink(0).Children(func() {
					kit.Spin(c, t.Tx3)
					ui.Text(c, "等 Claude 接收").FontSize(theme.FsCap).TextColor(t.Tx3)
				})
			})
		}
		for i, it := range a.q.items {
			i, it := i, it
			row := ui.Row(c).Key(it.ID).Height(26).PaddingX(6).Gap(8).AlignItems(ui.Center).Radius(t.RSm())
			if row.Hovered() {
				row.Background(t.Fill1)
			}
			_, imgs := splitDraftImages(it.Text)
			disp := queueShowText(it.Text)
			imgN := len(imgs)
			row.Children(func() {
				a.queueKind(c, false, i+1)
				ui.Text(c, disp).SingleLine().Grow(1).MinWidth(0).Cursor(ui.CursorPointer).Tooltip("点击退回输入框编辑").OnClick(func() {
					if s, ok := a.q.take(i); ok {
						a.draft = s
						a.ux.focusDraft = true
					}
				})
				if imgN > 0 {
					a.queueImgBadge(c, imgN)
				}
				if row.Hovered() {
					kit.Ib(c, "bolt", kit.IbOpt{Small: true, Tip: "改为引导:现在就插进当前回合"}).OnClick(func() {
						if s, ok := a.q.take(i); ok {
							a.steerSend(s)
						}
					})
					kit.Ib(c, "stop", kit.IbOpt{Small: true, Tip: "停止并发送"}).OnClick(func() { a.qStopAndSend(i) })
					kit.Ib(c, "trash", kit.IbOpt{Small: true, Tip: "删除"}).OnClick(func() { a.q.take(i) })
				}
			})
		}
	})
}

// queueImgBadge 行尾「N 张图」小标(设计稿 flow-queue)。
func (a *App) queueImgBadge(c *ui.Context, n int) {
	if n <= 0 {
		return
	}
	t := a.tk
	ui.Row(c).Gap(3).AlignItems(ui.Center).Shrink(0).Children(func() {
		kit.Ic(c, "image", kit.S12).TextColor(t.Tx3)
		ui.Text(c, itoa(n)+" 张图").FontSize(theme.FsCap).TextColor(t.Tx3).FontFeatures("tnum")
	})
}

// drainQueue 空闲时发出队首(暂停中不发);图片附件随正文一并解析发送。
func (a *App) drainQueue() {
	if a.st.Phase == core.PhaseRunning {
		return
	}
	if s, ok := a.q.next(); ok {
		a.submitPrompt(a.queueDraftFor(s), false)
	}
}

// permDiffPreviewMax 权限卡里最多直接画多少行差异,再多只给摘要(完整差异批准后在工作区审查)。
const permDiffPreviewMax = 14

// permDiffPreview 编辑 / 写入类权限请求:在卡片里预览将要发生的改动,让用户批准前看得到改什么。
func (a *App) permDiffPreview(c *ui.Context, r *core.PermissionRequest) {
	if r.Tool == nil || (r.Tool.Kind != core.KindEdit && r.Tool.Kind != core.KindWrite) {
		return
	}
	t := a.tk
	budget := permDiffPreviewMax
	hidden, hAdd, hDel := 0, 0, 0
	for _, ct := range r.Tool.Content {
		d, ok := ct.(*core.DiffBlock)
		if !ok {
			continue
		}
		n := len(diff.Lines(d.Old, d.New, d.Line, d.Line))
		if n > budget {
			ad, de := diff.Stat(diff.Lines(d.Old, d.New, d.Line, d.Line))
			hidden, hAdd, hDel = hidden+1, hAdd+ad, hDel+de
			continue
		}
		budget -= n
		a.diffCard(c, d, false)
	}
	if hidden > 0 {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "另有 "+itoa(hidden)+" 处较大改动未展开").FontSize(theme.FsCap).TextColor(t.Tx3)
			ui.Text(c, "+"+itoa(hAdd)).Font("monospace").FontSize(theme.FsCap).TextColor(t.Ok)
			ui.Text(c, "−"+itoa(hDel)).Font("monospace").FontSize(theme.FsCap).TextColor(t.Err)
		})
	}
}
