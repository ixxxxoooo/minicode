package app

import (
	"regexp"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diff"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// flowCol 是会话流的居中列(最宽 712,两侧 16)。
func (a *App) flowCol(c *ui.Context, inner func()) {
	col := ui.Column(c).MaxWidth(a.flowMaxWidth()).PaddingX(a.flowPad())
	if w, ok := a.flowSnapWidth(); ok {
		col.Width(w)
	} else {
		col.WidthPercent(100)
	}
	col.Children(inner)
}

// flowPad 是会话流内容列的左右内边距,按对话栏自己的宽度三档(设计稿 ws.css:宽 20 / 中 14 / 窄 10)。
func (a *App) flowPad() float32 {
	switch a.chatTier() {
	case tierWide:
		return 20
	case tierMid:
		return 14
	}
	return 10
}

// flowMaxWidth 是会话流与输入框等内容列的最大宽度(见 appearance.flowWidth 六档)。
func (a *App) flowMaxWidth() float32 {
	return a.flowContentMaxWidth()
}

// morphSnapStep 形态切换期间会话流内容宽度的吸附台阶(px)。
const morphSnapStep = 80

// snapDown 把宽度向下吸附到台阶的整数倍(不超过原宽,所以不会被裁)。
func snapDown(w, step float32) float32 {
	if w < step*2 {
		return w
	}
	return float32(int(w/step)) * step
}

// flowSnapWidth 形态切换(对话栏宽度做 440 ms 过渡)期间返回会话流内容列应使用的固定宽度。
//
// 库的 ElementTransition{Size} 会在每个中间宽度把整个子树重排;会话流没有虚拟化,
// 长会话(大项目里常见)里每条文字每帧都要重新断行,CoreText 断行每帧几十毫秒,动画只有 ~20 fps。
// 过渡期间内容本来就在淡入 / 淡出,这里把内容宽度吸附到 80px 台阶,
// 约 28 个中间宽度缩成约 5 个(各宽度的排版有全局缓存),过渡结束回到精确宽度。
func (a *App) flowSnapWidth() (float32, bool) {
	if a.win == nil || !kit.MotionOn() || !a.morphing() || a.chatW <= 0 {
		return 0, false
	}
	return snapDown(a.chatW, morphSnapStep), true
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
			if sa, ok := items[i].(*core.SubagentCard); ok {
				j := i
				var grp []*core.SubagentCard
				for j < len(items) {
					sa2, ok2 := items[j].(*core.SubagentCard)
					if !ok2 {
						break
					}
					grp = append(grp, sa2)
					j++
				}
				if len(grp) >= 2 {
					a.parallelSubGroup(c, grp)
					i = j
					continue
				}
				_ = sa
			}
			if _, ok := mergeable(items[i]); ok && a.ToolDetailLevel() == 0 {
				j := i
				var grp []*core.ToolCall
				for j < len(items) {
					tc2, ok2 := mergeable(items[j])
					if !ok2 {
						break
					}
					grp = append(grp, tc2)
					j++
				}
				if len(grp) >= 2 {
					a.mergeGroup(c, grp)
					i = j
					continue
				}
			}
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
	open := !a.prefs.Bool(pGroupFold, true)
	if set, ok := a.folds[id]; ok {
		open = set
	}
	sp := step{id: id, icon: "layers", verb: verb, obj: obj, fam: famExplore, state: state, hasBody: true, open: open}
	a.stepRow(c, sp, func() {
		a.nest(c, func() {
			for _, tc := range g {
				a.stepRow(c, a.presentTool(tc), nil)
			}
		})
	})
	if !open && running {
		if tc := lastOpenTool(g); tc != nil {
			a.groupLatest(c, id, tc)
		}
	}
}

func (a *App) stepItem(c *ui.Context, it core.Item) {
	switch v := it.(type) {
	case *core.Thinking:
		sp := step{id: v.ID, icon: "brain", verb: "思考", obj: thinkTitle(v.Text), fam: famThink, hasBody: strings.TrimSpace(v.Text) != "", open: a.prefs.Bool(pThinkOpen, false)}
		sp.thinkText = v.Text
		if v.Streaming {
			sp.state, sp.thinkLive = "run", true
			a.thinkWasLive[v.ID] = true
		} else {
			if a.thinkWasLive[v.ID] {
				a.noteThinkStreamEnd(v.ID)
			}
			if d, ok := a.st.Durations[v.ID]; ok && d > 0 {
				sp.right = core.FormatElapsed(int64(d) * 1000)
			}
		}
		a.stepRow(c, sp, func() { a.thinkBody(c, v.Text) })
	case *core.ToolCall:
		sp := a.presentTool(v)
		a.stepRow(c, sp, func() { a.toolBody(c, v) })
	case *core.PermissionRequest:
		if v.Tool != nil {
			sp := a.presentTool(v.Tool)
			denied := strings.Contains(strings.ToLower(v.Resolved), "deny") || strings.Contains(v.Resolved, "reject") || strings.Contains(v.Resolved, "拒绝")
			if denied {
				sp.state, sp.icon, sp.right = "deny", "lock", "已拒绝"
			}
			a.stepRow(c, sp, func() { a.toolBody(c, v.Tool) })
			if denied {
				a.permissionDenyNote(c, v.Tool)
			}
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
	if stepHasMenu(sp) { // 右键:复制路径 / 命令 / 输出 / 网址、在访达中显示、查看差异…(按这一行的类型)
		row.ContextMenu(func(m *ui.Menu) { a.stepMenu(m, sp) })
	}
	switch {
	case body != nil && sp.hasBody:
		row.OnClick(func() {
			// 会话流默认贴底:不拦的话,在底部展开会把整段内容向上顶,点的那一行跟着跑,
			// 鼠标不动再点就点到别的东西。HoldPlace 让这一帧不贴底,被点的行留在原地、内容向下长,
			// 再点一次(鼠标没动)就原地收起。
			a.flowList.HoldPlace()
			a.folds[sp.id] = !open
			a.foldAt[sp.id] = time.Now()
		})
	case sp.path != "": // 没有可展开内容的文件步骤(读取):点整行 = 在工作区打开文件
		row.Cursor(ui.CursorPointer).OnClick(func() {
			if !a.openPathRef(joinRef(sp.path, sp.line)) {
				a.toast = "文件已不存在:" + baseName(sp.path)
			}
		})
	case sp.term != nil: // 命令没有输出体时:点整行 = 在终端面板查看
		row.Cursor(ui.CursorPointer).OnClick(func() { a.viewInPanel(sp.term) })
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
			case sp.path != "": // 文件相关的步骤:显示文件 / 语言图标(按扩展名上色),点图标跳到文件
				a.fileIc(c, baseName(sp.path))
			default:
				kit.Ic(c, sp.icon, kit.S12).TextColor(t.Tx3.Mix(tone, .78))
			}
		})
		var obj ui.Element
		if sp.state == "run" && a.thinkMarqueeVisible(sp, open) {
			// 折叠的思考在流式输出:只留「思考」二字,右侧跑马灯占满整行(首句标题与预览重复,且会把预览挤到几个字宽)
			kit.RunTextS(c, sp.verb, t.Tx3, func(e ui.Element) ui.Element { return e.TextColor(t.Tx3).Shrink(0).SingleLine() })
			obj = ui.Box(c).Width(0).Shrink(0)
		} else if sp.state == "run" {
			kit.RunTextS(c, sp.verb, t.Tx3, func(e ui.Element) ui.Element { return e.TextColor(t.Tx3).Shrink(0).SingleLine() })
			obj = kit.RunTextS(c, sp.obj, t.Tx1, func(e ui.Element) ui.Element { return e.TextColor(t.Tx1).SingleLine().Shrink(1) })
		} else {
			// 字重规范(设计稿 v4):工具行的动词与对象都是常规字重,靠颜色分层——动词 Tx3 → 对象 Tx2 → 附注 Tx3 → 结果 Tx4;
			// 只有进行中的一行对象提亮到 Tx1。思考行整行更淡(对象也用 Tx3),不抢正文。
			ui.Text(c, sp.verb).TextColor(t.Tx3).Shrink(0).SingleLine()
			objCol := t.Tx2
			if sp.fam == famThink {
				objCol = t.Tx3
			}
			obj = ui.Text(c, sp.obj).TextColor(objCol).SingleLine().Shrink(1)
		}
		mq := a.thinkMarqueeRow(c, sp, tone, open)
		if at, ok := a.thinkMqHide[sp.id]; ok && kit.MotionOn() {
			age := float32(a.now().Sub(at)) / float32(kit.DurFast)
			if age < 1 {
				mq.Opacity(1 - age)
				a.frameSoon()
			}
		}
		if sp.state == "deny" {
			obj.Strikethrough().TextColor(t.Tx3)
		}
		if sp.meta != "" && a.chatTier() != tierNarrow { // 窄档:附注(路径、行区间)让位
			ui.Text(c, sp.meta).Font("monospace").FontSize(theme.Px(11)).TextColor(t.Tx3).SingleLine().Shrink(1)
		}
		ui.Spacer(c).Grow(1)
		// 右侧结果
		ui.Row(c).Gap(6).PaddingStart(8).Shrink(0).AlignItems(ui.Center).Children(func() {
			if hov && sp.state != "run" {
				a.stepActions(c, sp)
			}
			if sp.add > 0 || sp.del > 0 {
				if sp.add > 0 {
					ui.Text(c, "+"+itoa(sp.add)).Font("monospace").FontSize(theme.FsCap).TextColor(t.Ok)
				}
				if sp.del > 0 {
					ui.Text(c, "−"+itoa(sp.del)).Font("monospace").FontSize(theme.FsCap).TextColor(t.Err)
				}
			}
			if sp.right != "" && (a.chatTier() != tierNarrow || sp.state == "err") { // 窄档只留结果(错误仍显示)
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
		bx := ui.Column(c).Key("fold-"+sp.id).Margin(3, 0, 7, theme.FlowInset)
		// 展开入场:淡入并从上方 4px 落到位(设计稿 grow-y 的近似,高度不插值以免挤动下方内容)
		if at, ok := a.foldAt[sp.id]; ok && kit.MotionOn() {
			if p := float32(time.Since(at)) / float32(kit.DurBase); p < 1 {
				a.frameSoon()
				bx.Opacity(p).Translate(0, (p-1)*4)
			} else {
				delete(a.foldAt, sp.id)
			}
		}
		bx.Children(body)
	}
}

// nest 是嵌套的下一级(分组展开的子项):左侧一条细引导线。
func (a *App) nest(c *ui.Context, inner func()) {
	t := a.tk
	ui.Column(c).Margin(0, 0, 0, 7.5).Gap(1).Children(func() {
		// 行内 flex 子项的自动最小宽度 = 子树最小内容宽度:内层 Column 与外层 Row 都要显式 MinWidth(0),
		// 否则组里超长的命令 / 输出行会把所有卡片撑出会话流内容列(回合折叠「已处理」与探索组共用这里)。
		ui.Row(c).MinWidth(0).Children(func() {
			ui.Box(c).Width(1).Background(t.Bd2).Margin(0, 14, 0, 0).Shrink(0)
			ui.Column(c).Grow(1).MinWidth(0).Gap(1).Margin(0, 0, 0, -23).Children(inner)
		})
	})
}

// ───────── 展开内容 ─────────

func (a *App) thinkBody(c *ui.Context, text string) {
	t := a.tk
	ui.Row(c).Children(func() {
		ui.Box(c).Width(2).Background(t.Bd2).Margin(0, 8, 0, 0)
		ui.Text(c, strings.TrimSpace(text)).FontSize(a.FlowCapFont()).LineHeight(1.6).TextColor(t.Tx3).Grow(1)
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
	if !toolRichBody(tc) { // 没有差异 / 终端输出:统一给「入参 + 结果」详情,所有工具都能展开
		a.toolDetail(c, tc)
		return
	}
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
func (a *App) card(c *ui.Context, head func(), body func()) { a.cardM(c, head, nil, body) }

// cardM 是带头部右键菜单的卡片(差异卡 / 命令卡头部:复制、跳转;menu 为 nil 就没有右键菜单)。
func (a *App) cardM(c *ui.Context, head func(), menu func(m *ui.Menu), body func()) {
	a.cardHead(c, head, menu, body, false)
}

// cardHead 是 cardM 的底层:top 为真时头部允许多行(最小 24 高、随内容长高),右侧按钮顶部对齐;
// 否则头部固定 24 高、垂直居中。命令卡的命令按宽度换行用 top。
func (a *App) cardHead(c *ui.Context, head func(), menu func(m *ui.Menu), body func(), top bool) {
	t := a.tk
	bg := t.Sunken
	if t.Scheme.ID == "b" {
		bg = t.Raised
	}
	box := ui.Column(c).Background(bg).Radius(t.RMd()).Border(1, t.Bd1).Clip()
	a.cardHov = box.Hovered()
	box.Children(func() {
		if head != nil {
			// 注意:ui.Row(c) 一构造就进了父节点,不能先建一个再换另一个(多出来的空行就是一条空白标题栏)
			var hr ui.Element
			if top {
				hr = ui.Row(c).MinHeight(24).Padding(3, 4, 3, 9).Gap(6).AlignItems(ui.Start).TextColor(t.Tx3).FontSize(theme.FsCap).
					BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1)
			} else {
				hr = ui.Row(c).Height(theme.Px(24)).Padding(0, 4, 0, 9).Gap(6).AlignItems(ui.Center).TextColor(t.Tx3).FontSize(theme.FsCap).
					BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1)
			}
			if menu != nil {
				hr.ContextMenu(menu)
			}
			hr.Children(head)
		}
		body()
	})
}

const monoLH = 11.5 * 1.6 // IDE / 审查等工作区仍用固定行高

func (a *App) flowMonoLH() float32 { return a.FlowMonoSize() * 1.6 }

// diffCard 画差异卡;head 为真时带「src/router.rs +4 −1」文件头。
func (a *App) diffCard(c *ui.Context, d *core.DiffBlock, head bool) {
	t := a.tk
	ls := diff.Lines(d.Old, d.New, d.Line, d.Line)
	add, del := diff.Stat(ls)
	var h func()
	if head {
		h = func() {
			a.fileIc(c, baseName(d.Path))
			ui.Text(c, d.Path).Grow(1).SingleLine()
			ui.Text(c, "+"+itoa(add)).Font("monospace").FontSize(a.FlowMonoSize()).TextColor(t.Ok)
			ui.Text(c, "−"+itoa(del)).Font("monospace").FontSize(a.FlowMonoSize()).TextColor(t.Err)
			kit.Ib(c, "ext", kit.IbOpt{Small: true, Tip: "在工作区查看差异"}).OnClick(func() { a.OpenDiffTab(d.Path) })
			kit.Ib(c, "copy", kit.IbOpt{Small: true, Tip: "复制改动后的内容"}).OnClick(func() { a.copyOut(d.New, "改动后的内容") })
		}
	}
	var hm func(m *ui.Menu)
	if head { // 头部右键:在工作区查看差异 / 复制改动后的内容 / 复制路径
		hm = func(m *ui.Menu) {
			if menuItem(m, "查看差异").Chosen() {
				a.OpenDiffTab(d.Path)
			}
			m.Separator()
			if menuItem(m, "复制改动后的内容").Chosen() {
				a.copyOut(d.New, "改动后的内容")
			}
			if menuItem(m, "复制路径").Chosen() {
				a.copyOut(d.Path, "路径")
			}
		}
	}
	a.cardM(c, h, hm, func() {
		hov := a.cardHov
		ui.Column(c).Padding(3, 0).Children(func() {
			for i, l := range ls {
				a.diffLine(c, i, l)
			}
		})
		if !head { // 悬停时右上角出现「在工作区打开差异」(会话流里的差异只是预览,完整的差异在工作区)
			ui.Row(c).Absolute().Top(2).Right(3).Opacity(map[bool]float32{true: 1, false: 0}[hov]).Children(func() {
				kit.Ib(c, "ext", kit.IbOpt{Small: true, Tip: "在工作区打开差异"}).OnClick(func() { a.OpenDiffTab(d.Path) })
			})
		}
	})
}

// OpenDiffTab 在工作区开某个文件的差异标签(HEAD ↔ 工作区)。
func (a *App) OpenDiffTab(path string) {
	a.openDiffKind(dkHead, a.wsRel(path), true)
	a.revealWorkspace()
}

func (a *App) diffLine(c *ui.Context, i int, l diff.Line) {
	t := a.tk
	no := l.NewNo
	if l.Kind == diff.Del {
		no = l.OldNo
	}
	r := ui.Row(c).Key(i).Height(a.flowMonoLH()).AlignItems(ui.Center)
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
		ui.Text(c, itoa(no)).Font("monospace").FontSize(theme.Px(11.5)).TextColor(t.Tx4).TextAlign(ui.End).Width(30).PaddingEnd(8).Shrink(0)
		ui.RichText(c, kit.RustSpans(l.Text, t.Tx1, 11.5)...).NoWrap().PaddingStart(4).Grow(1).MinWidth(0).Clip()
	})
}

// deniedOutput 判断命令输出是不是「用户拒绝」的回执:本应用自己回的文案,
// 或终端版 claude 的 "The user doesn't want to proceed…"。
func deniedOutput(out string) bool {
	s := strings.TrimSpace(out)
	return s == "用户拒绝了此次操作" || strings.HasPrefix(s, "The user doesn't want to proceed")
}

func (a *App) termCard(c *ui.Context, tc *core.ToolCall, tb *core.TerminalBlock) {
	t := a.tk
	live := tb.ExitCode == nil
	a.cardHead(c, func() {
		kit.Ic(c, "prompt", kit.S12).Margin(3, 0, 0, 0).Shrink(0)
		// 命令按 shell 语法高亮(命令名 / 子命令 / 标志 / 字符串 / 变量 / 管道),按宽度换行;
		// 不含空格的超长词(长路径)没处可断,ClipX 兜底,不压到右侧按钮
		ui.RichText(c, a.shellSpans(tb.Command, a.FlowMonoSize())...).LineHeight(1.5).ClipX().Grow(1).MinWidth(0)
		if !live {
			if (tc != nil && tc.Status == core.ToolDenied) || deniedOutput(tb.Output) {
				// 用户拒绝了这条命令:根本没跑过,后端回的「退出码 1」只是拒绝信号,不当作失败显示
				// (历史回放没有权限事件,状态是 Failed,只能按拒绝文案识别)
				ui.Text(c, "未执行").TextColor(t.Tx3).Margin(3, 0, 0, 0).Shrink(0)
			} else if *tb.ExitCode == 0 {
				ui.Text(c, "通过").TextColor(t.Ok).Margin(3, 0, 0, 0).Shrink(0)
			} else {
				ui.Text(c, "退出码 "+itoa(*tb.ExitCode)).TextColor(t.Err).Margin(3, 0, 0, 0).Shrink(0)
			}
		}
		kit.Ib(c, "terminal", kit.IbOpt{Small: true, Tip: "在终端面板查看"}).OnClick(func() { a.viewInPanel(tb) })
		kit.Ib(c, "copy", kit.IbOpt{Small: true, Tip: "复制命令与输出"}).OnClick(func() {
			a.copyOut("$ "+tb.Command+"\n"+strings.TrimRight(tb.Output, "\n"), "命令与输出")
		})
	}, func(m *ui.Menu) { // 头部右键:在终端面板查看 / 复制命令 / 复制命令与输出
		if menuItem(m, "在终端面板查看").Chosen() {
			a.viewInPanel(tb)
		}
		m.Separator()
		if menuItem(m, "复制命令").Chosen() {
			a.copyOut(tb.Command, "命令")
		}
		if menuItem(m, "复制命令与输出").Chosen() {
			a.copyOut("$ "+tb.Command+"\n"+strings.TrimRight(a.liveTermOut(tc.ID, tb), "\n"), "命令与输出")
		}
	}, func() {
		out := a.liveTermOut(tc.ID, tb)
		a.foldBody(c, foldKey("term", tc.ID), out, live, true, func() {
			a.copyOut("$ "+tb.Command+"\n"+strings.TrimRight(out, "\n"), "命令与输出")
		})
	}, true)
}

func (a *App) textCard(c *ui.Context, text string) {
	key := foldKey("text", itoa(int(a.st.Rev))+":"+itoa(len(text)))
	a.card(c, nil, func() {
		a.foldBody(c, key, text, false, true, func() { a.copyOut(text, "输出") })
	})
}

// ───────── 提示 ─────────

var userTok = regexp.MustCompile("`([^`]+)`|@\\{([^}]+)\\}")

func (a *App) notice(c *ui.Context, n *core.Notice) {
	t := a.tk
	switch n.Level {
	case core.NoticeInfo:
		a.noteLine(c, "info", n.Text)
	default:
		bg, fg, icon := t.WarnSoft, t.Warn, "alert"
		if n.Level == core.NoticeError {
			bg, fg, icon = t.ErrSoft, t.Err, "errc"
		}
		ui.Row(c).Gap(8).AlignItems(ui.Start).Padding(7, 10).Radius(t.RMd()).Background(bg).Border(1, fg.Alpha(.22)).Children(func() {
			kit.Ic(c, icon, kit.S14).TextColor(fg).Margin(2, 0, 0, 0)
			ui.Text(c, n.Text).Grow(1).LineHeight(1.5)
			a.noticeActions(c, n)
		})
	}
}

func (a *App) noticeActions(c *ui.Context, n *core.Notice) {
	low := strings.ToLower(n.Text)
	switch n.Level {
	case core.NoticeError:
		ui.Column(c).Gap(4).Shrink(0).Children(func() {
			if a.st.Phase == core.PhaseRunning {
				kit.Btn(c, "重试", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "refresh"}).Opacity(0.4).Cursor(ui.CursorDefault)
			} else if kit.Btn(c, "重试", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "refresh"}).Clicked() {
				if uid := a.lastUserID(); uid != "" {
					if txt := a.userText(uid); txt != "" {
						a.send(core.CmdPrompt{Text: txt})
						log.Info("错误提示重试", "user", uid)
					}
				} else {
					a.toast = "没有可重试的提问"
				}
			}
			if kit.Btn(c, "复制", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "copy"}).Clicked() {
				a.copyOut(n.Text, "错误信息")
			}
		})
	case core.NoticeWarning:
		if strings.Contains(n.Text, "压缩") || strings.Contains(low, "context") || strings.Contains(n.Text, "%") {
			kit.Btn(c, "压缩", kit.BtnOpt{Small: true, Variant: kit.BtnSoft}).OnClick(func() {
				a.send(core.CmdCompact{})
				log.Info("提示卡压缩上下文")
			})
		}
	}
}
