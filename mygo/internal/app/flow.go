package app

import (
	"regexp"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diff"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// flowCol 是会话流的居中列(最宽 712,两侧 16)。
func (a *App) flowCol(c *ui.Context, inner func()) {
	pad := float32(16)
	if a.mode == ModeIDE {
		pad = 12
	}
	ui.Column(c).WidthPercent(100).MaxWidth(theme.WCol).PaddingX(pad).Children(inner)
}

// flow 渲染整条时间线:用户消息、步骤组(思考 / 工具 / 待办 / 子 Agent)、助手正文、回合尾部。
func (a *App) flow(c *ui.Context) {
	items := a.st.Items
	ui.Column(c).Gap(8).Padding(12, 0, 18, 0).Children(func() {
		for i := 0; i < len(items); {
			last := i
			if isStepItem(items[i]) {
				j := i
				for j < len(items) && isStepItem(items[j]) {
					j++
				}
				a.stepGroup(c, items[i:j])
				last = j - 1
			} else {
				switch v := items[i].(type) {
				case *core.User:
					a.userMsg(c, v)
				case *core.Assistant:
					a.assistant(c, v)
				case *core.Notice:
					a.notice(c, v)
				case *core.PlanApproval:
					a.planCard(c, v)
				case *core.QuestionRequest:
					if v.Answered {
						a.askEcho(c, v)
					}
				}
			}
			// 回合尾部操作条:每个已结束回合的最后一项之后(进行中的回合还没有统计,自然不显示)
			if uid, ok := turnEnd(items, last); ok {
				a.turnFooter(c, uid)
			}
			i = last + 1
		}
	})
}

// isStepItem 属于「步骤行」的条目(会收进一个活动组)。
func isStepItem(it core.Item) bool {
	switch it.(type) {
	case *core.Thinking, *core.ToolCall, *core.Todos, *core.SubagentCard:
		return true
	case *core.PermissionRequest:
		return it.(*core.PermissionRequest).Resolved != ""
	}
	return false
}

// stepGroup 是一串连续的步骤(设计稿 .acts-g):行间距 1;方案 B 收进一张凹槽底圆角卡;方案 C 串在一条时间轴上。
func (a *App) stepGroup(c *ui.Context, items []core.Item) {
	t := a.tk
	g := ui.Column(c).Gap(1)
	switch t.Scheme.ID {
	case "b":
		g.Background(t.Sunken).Radius(t.RMd()).Padding(3, 10).Border(1, t.Bd1)
	}
	g.Children(func() {
		if t.Scheme.ID == "c" {
			// 时间轴竖线:贴着图标列中线
			ui.Box(c).Absolute().Left(7.5).Top(10).Bottom(10).Width(1).Background(t.Bd2)
		}
		for i := 0; i < len(items); {
			if tc, ok := isExplore(items[i]); ok {
				_ = tc
				j := i
				var grp []*core.ToolCall
				for j < len(items) {
					tc2, ok2 := isExplore(items[j])
					if !ok2 {
						break
					}
					grp = append(grp, tc2)
					j++
				}
				if len(grp) >= 2 {
					a.exploreGroup(c, grp)
					i = j
					continue
				}
			}
			a.stepItem(c, items[i])
			i++
		}
	})
}

func (a *App) exploreGroup(c *ui.Context, g []*core.ToolCall) {
	sum, running := exploreSummary(g)
	id := "grp:" + g[0].ID
	verb, obj := "探索了", sum
	state := ""
	if running {
		verb, obj, state = "正在探索", "项目代码", "run"
	}
	sp := step{id: id, icon: "layers", verb: verb, obj: obj, fam: famExplore, state: state, hasBody: true}
	a.stepRow(c, sp, func() {
		a.nest(c, func() {
			for _, tc := range g {
				a.stepRow(c, presentTool(tc), nil)
			}
		})
	})
}

func (a *App) stepItem(c *ui.Context, it core.Item) {
	switch v := it.(type) {
	case *core.Thinking:
		sp := step{id: v.ID, icon: "brain", verb: "思考", obj: thinkTitle(v.Text), fam: famThink, hasBody: strings.TrimSpace(v.Text) != ""}
		if v.Streaming {
			sp.state = "run"
		} else if d, ok := a.st.Durations[v.ID]; ok && d > 0 {
			sp.right = core.FormatElapsed(int64(d) * 1000)
		}
		a.stepRow(c, sp, func() { a.thinkBody(c, v.Text) })
	case *core.ToolCall:
		sp := presentTool(v)
		a.stepRow(c, sp, func() { a.toolBody(c, v) })
	case *core.PermissionRequest:
		if v.Tool != nil {
			sp := presentTool(v.Tool)
			if strings.Contains(strings.ToLower(v.Resolved), "deny") || strings.Contains(v.Resolved, "reject") {
				sp.state, sp.icon = "deny", "lock"
			}
			a.stepRow(c, sp, func() { a.toolBody(c, v.Tool) })
		}
	case *core.Todos:
		done, total := 0, len(v.Items)
		for _, td := range v.Items {
			if td.Status == core.TodoCompleted {
				done++
			}
		}
		sp := step{id: v.ID, icon: "todo", verb: "待办", obj: itoa(done) + " / " + itoa(total) + " 完成", fam: famPlan, hasBody: true}
		a.stepRow(c, sp, func() { a.todoBody(c, v) })
	case *core.SubagentCard:
		a.subagentCard(c, v)
	}
}

// ───────── 步骤行 ─────────

// stepRow 画一行步骤。body 非 nil 时行可展开(点击切换,默认展开状态取 sp.open)。
func (a *App) stepRow(c *ui.Context, sp step, body func()) {
	t := a.tk
	open, set := a.folds[sp.id]
	if !set {
		open = sp.open
	}
	row := ui.ButtonBase(c).Key("step-"+sp.id).MinHeight(24).Margin(0, -8).PaddingX(8).Gap(6).Radius(t.RSm()).AlignItems(ui.Center).
		TextColor(t.Tx2).Cursor(ui.CursorDefault).Transition(kit.Fade)
	hov := row.Hovered()
	if hov {
		row.Background(t.Fill1)
	}
	if body != nil && sp.hasBody {
		row.OnClick(func() { a.folds[sp.id] = !open })
	}
	tone := a.famColor(sp.fam)
	row.Children(func() {
		// 图标列
		ic := ui.Row(c).Size(16, 16).Justify(ui.Center).AlignItems(ui.Center).Shrink(0)
		ic.Children(func() {
			switch {
			case sp.state == "run":
				kit.Spin(c, tone)
			case sp.state == "err":
				kit.Ic(c, sp.icon, kit.S12).TextColor(t.Err)
			default:
				kit.Ic(c, sp.icon, kit.S12).TextColor(t.Tx3.Mix(tone, .78))
			}
		})
		verb := ui.Text(c, sp.verb).TextColor(t.Tx3).Shrink(0).SingleLine()
		obj := ui.Text(c, sp.obj).FontWeight(500).TextColor(t.Tx1).SingleLine().Shrink(1)
		if sp.state == "run" { // 运行中的步骤文字呼吸
			kit.Pulse(verb)
			kit.Pulse(obj)
		}
		if sp.state == "deny" {
			obj.Strikethrough().TextColor(t.Tx3)
		}
		if sp.meta != "" {
			ui.Text(c, sp.meta).Font("monospace").FontSize(11).TextColor(t.Tx3).SingleLine().Shrink(1)
		}
		ui.Spacer(c).Grow(1)
		// 右侧结果
		ui.Row(c).Gap(6).PaddingStart(8).Shrink(0).AlignItems(ui.Center).Children(func() {
			if sp.add > 0 || sp.del > 0 {
				if sp.add > 0 {
					ui.Text(c, "+"+itoa(sp.add)).Font("monospace").FontSize(theme.FsCap).TextColor(t.Ok)
				}
				if sp.del > 0 {
					ui.Text(c, "−"+itoa(sp.del)).Font("monospace").FontSize(theme.FsCap).TextColor(t.Err)
				}
			}
			if sp.right != "" {
				col := t.Tx4
				if sp.state == "err" {
					col = t.Err
				}
				ui.Text(c, sp.right).FontSize(theme.FsCap).TextColor(col).FontFeatures("tnum").SingleLine()
			}
			if body != nil && sp.hasBody {
				ch := kit.Ic(c, "chevR", kit.S12).TextColor(t.Tx4)
				rot := float32(0)
				op := float32(0)
				if open {
					rot, op = 90, 1
				} else if hov {
					op = 1
				}
				ch.Rotate(ch.AnimateWith("rot", rot, kit.DurBase, kit.EaseOut)).Opacity(ch.AnimateWith("op", op, kit.DurFast, kit.EaseOut))
			}
		})
	})
	if body != nil && sp.hasBody && open {
		ui.Column(c).Key("fold-"+sp.id).Margin(3, 0, 7, theme.FlowInset).Children(body)
	}
}

// nest 是嵌套的下一级(分组展开的子项):左侧一条细引导线。
func (a *App) nest(c *ui.Context, inner func()) {
	t := a.tk
	ui.Column(c).Margin(0, 0, 0, 7.5).Gap(1).Children(func() {
		ui.Row(c).Children(func() {
			ui.Box(c).Width(1).Background(t.Bd2).Margin(0, 14, 0, 0)
			ui.Column(c).Grow(1).Gap(1).Margin(0, 0, 0, -23).Children(inner)
		})
	})
}

// ───────── 展开内容 ─────────

func (a *App) thinkBody(c *ui.Context, text string) {
	t := a.tk
	ui.Row(c).Children(func() {
		ui.Box(c).Width(2).Background(t.Bd2).Margin(0, 8, 0, 0)
		ui.Text(c, strings.TrimSpace(text)).FontSize(12).LineHeight(1.6).TextColor(t.Tx3).Grow(1)
	})
}

func (a *App) todoBody(c *ui.Context, td *core.Todos) {
	t := a.tk
	ui.Column(c).Gap(5).Padding(2, 0).Children(func() {
		for _, it := range td.Items {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				kit.Chk(c, int(it.Status))
				x := ui.Text(c, it.Content)
				switch it.Status {
				case core.TodoCompleted:
					x.TextColor(t.Tx3).Strikethrough().DecorationColor(t.Tx4)
				case core.TodoInProgress:
					x.TextColor(t.Tx1).FontWeight(500)
				default:
					x.TextColor(t.Tx2)
				}
			})
		}
	})
}

func (a *App) toolBody(c *ui.Context, tc *core.ToolCall) {
	for _, ct := range tc.Content {
		switch v := ct.(type) {
		case *core.DiffBlock:
			a.diffCard(c, v, false)
		case *core.TerminalBlock:
			a.termCard(c, tc, v)
		case *core.TextBlock:
			if tc.Kind != core.KindRead && tc.Kind != core.KindSearch {
				a.textCard(c, v.Text)
			}
		}
	}
}

// card 是工具卡:凹槽底 + 描边 + 24 高的头。
func (a *App) card(c *ui.Context, head func(), body func()) {
	t := a.tk
	bg := t.Sunken
	if t.Scheme.ID == "b" {
		bg = t.Raised
	}
	ui.Column(c).Background(bg).Radius(t.RMd()).Border(1, t.Bd1).Clip().Children(func() {
		if head != nil {
			ui.Row(c).Height(24).Padding(0, 4, 0, 9).Gap(6).AlignItems(ui.Center).TextColor(t.Tx3).FontSize(theme.FsCap).
				BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(head)
		}
		body()
	})
}

const monoLH = 11.5 * 1.6

// diffCard 画差异卡;head 为真时带「src/router.rs +4 −1」文件头。
func (a *App) diffCard(c *ui.Context, d *core.DiffBlock, head bool) {
	t := a.tk
	ls := diff.Lines(d.Old, d.New, d.Line, d.Line)
	add, del := diff.Stat(ls)
	var h func()
	if head {
		h = func() {
			kit.Ic(c, "diff", kit.S12)
			ui.Text(c, d.Path).Grow(1).SingleLine()
			ui.Text(c, "+"+itoa(add)).Font("monospace").FontSize(11.5).TextColor(t.Ok)
			ui.Text(c, "−"+itoa(del)).Font("monospace").FontSize(11.5).TextColor(t.Err)
			kit.Ib(c, "copy", kit.IbOpt{Small: true, Tip: "复制"})
		}
	}
	a.card(c, h, func() {
		ui.Column(c).Padding(3, 0).Children(func() {
			for i, l := range ls {
				a.diffLine(c, i, l)
			}
		})
	})
}

func (a *App) diffLine(c *ui.Context, i int, l diff.Line) {
	t := a.tk
	no := l.NewNo
	if l.Kind == diff.Del {
		no = l.OldNo
	}
	r := ui.Row(c).Key(i).Height(monoLH).AlignItems(ui.Center)
	switch l.Kind {
	case diff.Add:
		r.Background(t.DiffAdd)
	case diff.Del:
		r.Background(t.DiffDel)
	}
	r.Children(func() {
		switch l.Kind {
		case diff.Add:
			ui.Box(c).Absolute().Left(0).Top(0).Bottom(0).Width(2).Background(t.Ok)
		case diff.Del:
			ui.Box(c).Absolute().Left(0).Top(0).Bottom(0).Width(2).Background(t.Err)
		}
		ui.Text(c, itoa(no)).Font("monospace").FontSize(11.5).TextColor(t.Tx4).TextAlign(ui.End).Width(30).PaddingEnd(8).Shrink(0)
		ui.RichText(c, kit.RustSpans(l.Text, t.Tx1, 11.5)...).NoWrap().PaddingStart(4).Grow(1).MinWidth(0).Clip()
	})
}

var (
	termHead = regexp.MustCompile(`^(\s*(?:Compiling|Running|Finished|Downloaded|Checking|Fresh|Blocking|Updating))(.*)$`)
	termOK   = regexp.MustCompile(`\bok\b`)
	termIn   = regexp.MustCompile(`\bin \d+(?:\.\d+)?s\b`)
)

// termLine 把一行终端输出切成带色片段。
func (a *App) termLine(line string) []ui.Span {
	t := a.tk
	mk := func(s string, col ui.Color) ui.Span {
		return ui.Span{Text: s, Font: "monospace", Size: 11.5, Color: col}
	}
	if m := termHead.FindStringSubmatch(line); m != nil {
		return append([]ui.Span{mk(m[1], t.Tx3)}, a.termRest(m[2], mk)...)
	}
	return a.termRest(line, mk)
}

func (a *App) termRest(s string, mk func(string, ui.Color) ui.Span) []ui.Span {
	t := a.tk
	type mark struct {
		i, j int
		col  ui.Color
	}
	var ms []mark
	for _, m := range termOK.FindAllStringIndex(s, -1) {
		ms = append(ms, mark{m[0], m[1], t.Ok})
	}
	for _, m := range termIn.FindAllStringIndex(s, -1) {
		ms = append(ms, mark{m[0], m[1], t.Tx3})
	}
	// 按起点排序(两类各自有序,数量极少,插入排序即可)
	for i := 1; i < len(ms); i++ {
		for j := i; j > 0 && ms[j].i < ms[j-1].i; j-- {
			ms[j], ms[j-1] = ms[j-1], ms[j]
		}
	}
	var out []ui.Span
	last := 0
	for _, m := range ms {
		if m.i < last {
			continue
		}
		if m.i > last {
			out = append(out, mk(s[last:m.i], t.Tx1))
		}
		out = append(out, mk(s[m.i:m.j], m.col))
		last = m.j
	}
	if last < len(s) {
		out = append(out, mk(s[last:], t.Tx1))
	}
	if len(out) == 0 {
		out = append(out, mk(" ", t.Tx1))
	}
	return out
}

func (a *App) termCard(c *ui.Context, tc *core.ToolCall, tb *core.TerminalBlock) {
	t := a.tk
	live := tb.ExitCode == nil
	a.card(c, func() {
		kit.Ic(c, "prompt", kit.S12)
		ui.Text(c, tb.Command).Font("monospace").FontSize(11.5).Grow(1).SingleLine()
		if !live {
			if *tb.ExitCode == 0 {
				ui.Text(c, "通过").TextColor(t.Ok)
			} else {
				ui.Text(c, "退出码 "+itoa(*tb.ExitCode)).TextColor(t.Err)
			}
		}
		kit.Ib(c, "copy", kit.IbOpt{Small: true, Tip: "复制"})
	}, func() {
		ui.Column(c).Padding(6, 10).Children(func() {
			lines := strings.Split(strings.TrimRight(tb.Output, "\n"), "\n")
			for i, ln := range lines {
				ui.Row(c).Key(i).Height(monoLH).AlignItems(ui.Center).Children(func() {
					ui.RichText(c, a.termLine(ln)...).NoWrap().Clip()
					if live && i == len(lines)-1 {
						kit.Caret(c)
					}
				})
			}
		})
	})
}

func (a *App) textCard(c *ui.Context, text string) {
	t := a.tk
	a.card(c, nil, func() {
		ui.Column(c).Padding(6, 10).Children(func() {
			for i, ln := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
				if i >= 12 {
					break
				}
				ui.Text(c, ln).Key(i).Font("monospace").FontSize(11.5).TextColor(t.Tx1).NoWrap()
			}
		})
	})
}

// ───────── 用户消息 / 助手正文 / 提示 ─────────

var userTok = regexp.MustCompile("`([^`]+)`|@\\{([^}]+)\\}")

// userMsg 是用户消息卡:行内代码与 @{文件} 引用标记。
func (a *App) userMsg(c *ui.Context, u *core.User) {
	t := a.tk
	card := ui.Column(c).Background(t.Raised).Radius(t.RMd()).Padding(7, 11)
	switch t.Scheme.ID {
	case "b":
		card.AlignSelf(ui.End).MaxWidthPercent(86).Background(t.AcSoft).Radius(t.RLg(), t.RLg(), t.RXs(), t.RLg())
	case "c":
		card.Background(t.Fill1).Radius(0, t.RSm(), t.RSm(), 0)
		card.Border(0, ui.Transparent)
	default:
		for _, s := range t.Shadow1() {
			card.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
		}
	}
	card.Children(func() {
		if t.Scheme.ID == "c" {
			ui.Box(c).Absolute().Left(0).Top(0).Bottom(0).Width(2).Background(t.Ac)
		}
		var spans []ui.Span
		txt := u.Text
		last := 0
		for _, m := range userTok.FindAllStringSubmatchIndex(txt, -1) {
			if m[0] > last {
				spans = append(spans, ui.Span{Text: txt[last:m[0]]})
			}
			if m[2] >= 0 {
				spans = append(spans, ui.Span{Text: txt[m[2]:m[3]], Font: "monospace", Size: 12, Background: t.Fill2, Radius: t.RXs(), PaddingX: 3})
			} else {
				spans = append(spans, ui.Span{Text: " " + baseName(txt[m[4]:m[5]]) + " ", Size: 11.5, Color: t.Info, Background: t.InfoSoft, Radius: t.RXs(), PaddingX: 4})
			}
			last = m[1]
		}
		if last < len(txt) {
			spans = append(spans, ui.Span{Text: txt[last:]})
		}
		ui.RichText(c, spans...).FontSize(theme.FsRead).LineHeight(1.55).TextColor(t.Tx1)
	})
}

func (a *App) notice(c *ui.Context, n *core.Notice) {
	t := a.tk
	switch n.Level {
	case core.NoticeInfo:
		ui.Row(c).MinHeight(22).Gap(6).TextColor(t.Tx3).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Width(16).Justify(ui.Center).Children(func() { kit.Ic(c, "info", kit.S12) })
			ui.Text(c, n.Text)
		})
	default:
		bg, fg, icon := t.WarnSoft, t.Warn, "alert"
		if n.Level == core.NoticeError {
			bg, fg, icon = t.ErrSoft, t.Err, "errc"
		}
		ui.Row(c).Gap(8).AlignItems(ui.Start).Padding(7, 10).Radius(t.RMd()).Background(bg).Border(1, fg.Alpha(.22)).Children(func() {
			kit.Ic(c, icon, kit.S14).TextColor(fg).Margin(2, 0, 0, 0)
			ui.Text(c, n.Text).Grow(1).LineHeight(1.5)
		})
	}
}
