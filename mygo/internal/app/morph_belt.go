// Agent ⇄ IDE 切换的「传送带」平移(设计稿 app.js beltPrepare / interact.css「形态切换的编排」):
// 旧工作区 | 对话栏 | 新工作区 是刚性的一串,整体平移同一段距离(= 对话栏自己的位移)。
//
//	IDE → Agent:对话栏从右侧停靠移到中间,旧编辑区被挤出左缘,工作区自右缘移入;
//	Agent → IDE:反过来,向右。
//
// 切换那一帧记下旧几何,之后每帧按「旧几何 → 当前终态几何」用同一个缓动进度逐帧摆位;
// 三块面板都用绝对定位 + 固定宽度,宽度只有对话栏在变,动画结束后无缝交还给弹性布局(两边公式一致,不会在最后一帧跳宽)。
//
// @author ixxxxoooo
package app

import (
	"os"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// beltRect 是一块面板在舞台里的横向位置与宽度(布局像素)。
type beltRect struct{ x, w float32 }

// beltGeom 是某个形态下对话栏与工作区的几何。
type beltGeom struct {
	chat, ws     beltRect
	chatOn, wsOn bool // 对话栏 / 工作区此刻占位(不占位时宽度为 0)

	// 以下只在切换那一刻(beltCapture)记下,供被挤出舞台的旧工作区「幽灵」画内容:
	ghostSt *editor.State // 当前文件标签的只读影子编辑器(与真实编辑器共用缓冲区,滚动位置同切换那一刻);nil = 主体留空
	ghostV  *ideTab       // 当前是虚拟标签(审查 / 计划 / 子 Agent / 任务 …)时记下它,幽灵直接再画一份它的视图
	empty   bool          // 没有任何标签(工作区显示启动页 wsEmpty):幽灵再画一份启动页
	panelH  float32       // 旧工作区底部面板的高度(没开 = 0)
	term    []string      // 底部面板当前终端画面的最后几行文字(幽灵里的终端预览,纯文字)
}

// beltFrame 是本帧传送带的摆位结果;on = false 表示不在平移中,面板走弹性布局。
type beltFrame struct {
	on         bool
	chat, ws   beltRect // 对话栏与新工作区
	ghost      beltRect // 被挤出舞台的旧工作区
	chatOn     bool
	wsOn, ghOn bool
}

// morphFreezeP 是调试用:环境变量 MINI_MORPH_P(0..1)把形态切换冻结在这个进度上,
// 配合 cmd/shot 的 -click IDE 就能截到平移途中的画面(截图没有真实窗口与时钟)。
var morphFreezeP = func() float32 {
	v, err := strconv.ParseFloat(os.Getenv("MINI_MORPH_P"), 32)
	if err != nil || v <= 0 || v >= 1 {
		return 0
	}
	return float32(v)
}()

// beltGeomNow 按当前形态与开合状态,算对话栏与工作区在舞台里的几何。
// 公式与 chatPane / wsPane 的弹性布局一一对应:Agent = 对话占满、工作区固定宽在右;IDE = 对话固定宽在右、工作区占满。
// 参数 s:舞台宽度;gap:面板间距。
func (a *App) beltGeomNow(s, gap float32) beltGeom {
	g := beltGeom{}
	g.chatOn = a.mode == ModeAgent || a.chatMax || (a.rightOn && !a.useDockRail())
	g.wsOn = a.wsOccupies()
	if a.mode == ModeAgent {
		g.chat.w = s
		if g.wsOn {
			g.ws.w = a.wsRW()
			g.chat.w = max(0, s-g.ws.w-gap)
			g.ws.x = g.chat.w + gap
		}
		return g
	}
	switch {
	case a.chatMax:
		g.chat.w = s
	case g.chatOn:
		g.chat.w = a.dockW()
	}
	if g.wsOn {
		g.ws.w = s
		if g.chatOn {
			g.ws.w = max(0, s-g.chat.w-gap)
		}
	}
	if g.chatOn && !a.chatMax {
		g.chat.x = g.ws.w + gap
		if !g.wsOn {
			g.chat.x = 0
		}
	}
	return g
}

// beltCapture 在切形态的那一刻(模式与开合还是旧的)记下旧几何;舞台宽度未知时不记,走原有的弹性布局过渡。
func (a *App) beltCapture() {
	a.belt0Ok = false
	if a.stageW <= 0 {
		return
	}
	g := a.beltGeomNow(a.stageW, a.tk.Scheme.GapPane)
	if g.wsOn {
		g.ghostSt, g.panelH = a.beltGhostState()
		if tb := a.ide.cur(); g.ghostSt == nil && tb != nil && tb.virt != "" && a.ide.splitDir == 0 && !(a.ly.panelMax && a.ide.bottomOpen) {
			g.ghostV = tb
		}
		g.empty = a.ide.cur() == nil && a.ide.splitDir == 0 && !(a.ly.panelMax && a.ide.bottomOpen)
		if g.panelH > 0 {
			g.term = a.ghostTermLines(9)
		}
	}
	a.belt0, a.belt0Ok = g, true
}

// beltGhostState 给旧工作区的幽灵备好内容:当前是普通文件标签时,另建一个只读的编辑器状态,
// 与真实编辑器共用同一份缓冲区(语法高亮随缓冲区走),滚动位置取切换那一刻的。
// 真实的编辑器 / 终端不能在一帧里画两份(句柄只能绑一个元素),影子状态是独立的,不冲突。
// 返回:影子状态(其它标签类型为 nil,主体留空)、底部面板高度(没开 / 已最大化为 0)。
func (a *App) beltGhostState() (*editor.State, float32) {
	d := a.ide
	var panelH float32
	if d.bottomOpen && !a.ly.panelMax {
		panelH = a.panelHeight()
	}
	tb := d.cur()
	if tb == nil || tb.buf == nil || tb.st == nil || tb.diff || tb.virt != "" || tb.viewer != "" || tb.err != "" || d.splitDir != 0 || (a.ly.panelMax && d.bottomOpen) {
		return nil, panelH
	}
	st := &editor.State{Buffer: tb.buf, ReadOnly: true, WrapGuides: nil, NoScrollbars: true}
	st.SetScrollY(tb.st.ScrollY())
	return st, panelH
}

// beltStep 算本帧的摆位。不在平移中(没有真实窗口且没冻结 / 动效关 / 没记到旧几何 / 已走完)返回 on = false。
func (a *App) beltStep() beltFrame {
	if (a.win == nil && !a.driveFrames && morphFreezeP == 0) || (a.screen != ScreenWork && a.screen != ScreenWelcome) || !a.belt0Ok || a.stageW <= 0 || a.morphAt.IsZero() {
		return beltFrame{}
	}
	e, done := a.morphProgress()
	if morphFreezeP > 0 {
		e, done = morphFreezeP, false
	}
	if done {
		a.belt0Ok = false
		return beltFrame{}
	}
	gap := a.tk.Scheme.GapPane
	g0, g1 := a.belt0, a.beltGeomNow(a.stageW, gap)
	dx := g1.chat.x - g0.chat.x // 带子的位移:IDE → Agent 为负(向左),Agent → IDE 为正(向右)
	lerp := func(p, q float32) float32 { return p + (q-p)*e }
	f := beltFrame{on: true, chatOn: g0.chatOn || g1.chatOn, wsOn: g1.wsOn && g1.ws.w > .5, ghOn: g0.wsOn && g0.ws.w > .5}
	f.chat = beltRect{x: lerp(g0.chat.x, g1.chat.x), w: lerp(g0.chat.w, g1.chat.w)}
	f.ws = beltRect{x: g1.ws.x - dx*(1-e), w: g1.ws.w} // 新工作区:自与对话栏相邻的那一侧移进来
	f.ghost = beltRect{x: g0.ws.x + dx*e, w: g0.ws.w}  // 旧工作区:被对话栏挤出舞台
	// 对话栏的宽度在切换里也在变(Agent 居中占满 ↔ IDE 窄栏),三块并不是刚性平移同一段距离:
	// 若新旧工作区都照「平移 dx」摆,对话栏变窄 / 变宽的那一段就会在它与旧工作区(或新工作区)之间留一条空白
	// (实测最大 140 px,整个切换里都在)。改成「首尾相接的一串」:新 / 旧工作区各自紧贴对话栏的外缘(隔一个面板间距),
	// 对话栏自己的位置与宽度仍按进度插值。两端落点与上面的刚性公式一致(在起点与终点都严格重合),只是中间不再有缝。
	if g0.chatOn && g1.chatOn && g0.chat.w > .5 && g1.chat.w > .5 {
		right := f.chat.x + f.chat.w + gap // 对话栏右缘 + 间距:右侧那块面板的左缘
		left := func(w float32) float32 { return f.chat.x - gap - w }
		if a.mode == ModeIDE { // Agent → IDE:新工作区在对话栏左边,旧工作区在右边
			f.ws.x, f.ghost.x = left(f.ws.w), right
		} else { // IDE → Agent:旧工作区在左边,新工作区在右边
			f.ghost.x, f.ws.x = left(f.ghost.w), right
		}
	}
	return f
}

// wsGhost 画被挤出舞台的旧工作区。
// 同一份工作区内容(编辑器 / 终端等控件的句柄)一帧里只能绑定一个元素,不能再画第二份,
// 所以幽灵是轻量版:面板外壳 + 标签条(标签名,当前标签高亮)+ 主体 + 底部面板占位,不接收输入(切换期间整窗有指针罩)。
// 主体:当前是普通文件标签时画只读影子编辑器(真实代码 + 行号 + 高亮,不是一片空白);其它标签类型(对比 / 查看器 /
// 虚拟标签)只画底色。
func (a *App) wsGhost(c *ui.Context, r beltRect) {
	t := a.tk
	g := a.belt0
	p := a.pane(c).Key("ws-ghost").ClipX().Absolute().Left(r.x).Top(0).Bottom(0).Width(r.w).Shrink(0).Transition(noMotion).PassThrough()
	a.probe(pzGhost, p)
	p.Children(func() {
		ui.Row(c).Key("ws-ghost-tabs").Height(theme.HTab).Shrink(0).AlignItems(ui.Center).Padding(0, 6, 0, 6).Gap(2).
			Background(t.Chrome).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
			for _, tb := range a.ide.tabs {
				x := ui.Text(c, tb.title()).SingleLine().FontSize(theme.Px(11.5)).Padding(0, 8, 0, 8).TextColor(t.Tx3)
				if tb.id == a.ide.active {
					x.TextColor(t.Tx1).FontWeight(500)
				}
			}
		})
		if st := g.ghostSt; st != nil {
			st.Theme = a.editorTheme()
			st.Font = ui.Font{Family: "monospace", Size: 12}
			st.LineHeight = 20.0 / 12.0
			ui.Column(c).Key("ghost-body").Grow(1).MinHeight(0).Clip().PassThrough().Children(func() {
				editor.View(c.Key("ghost-ed"), st).Grow(1).MinWidth(0).FillHeight()
			})
		} else if tb := g.ghostV; tb != nil {
			ui.Column(c).Key("ghost-body").Grow(1).MinHeight(0).Clip().PassThrough().Children(func() {
				// 同一份视图画第二份:个别视图若有「只能绑一次」的控件会 panic,这里隔离,最坏只是幽灵主体留空(静默,5 秒后重试)
				a.guard(c, "形态切换幽灵", "belt-ghost:"+tb.id, guardSilent, func() { a.virtView(c, tb) })
			})
		} else if g.empty {
			ui.Column(c).Key("ghost-body").Grow(1).MinHeight(0).Clip().PassThrough().Children(func() {
				a.guard(c, "形态切换幽灵", "belt-ghost:empty", guardSilent, func() { a.wsEmpty(c) })
			})
		} else {
			ui.Box(c).Key("ghost-body").Grow(1)
		}
		if g.panelH > 0 {
			a.ghostPanel(c, g)
		}
	})
}

// ghostPanel 画幽灵里的底部面板:标签条(终端 / 问题 / 输出)+ 终端画面最后几行文字。
// 真实终端控件的句柄只能绑一个元素,所以这里是静态预览。
func (a *App) ghostPanel(c *ui.Context, g beltGeom) {
	t := a.tk
	ui.Column(c).Key("ghost-panel").Height(g.panelH).Shrink(0).ClipX().Background(t.BG).BorderWidth(1, 0, 0, 0).BorderColor(t.Bd2).Children(func() {
		ui.Row(c).Key("ghost-ptabs").Height(theme.HTab).Shrink(0).AlignItems(ui.Center).Padding(0, 8, 0, 8).Gap(4).Children(func() {
			for i, s := range []string{"终端", "问题", "输出"} {
				x := ui.Text(c, s).SingleLine().FontSize(theme.Px(11.5)).Padding(0, 6, 0, 6).TextColor(t.Tx3)
				if i == 0 {
					x.TextColor(t.Tx1).FontWeight(500)
				}
			}
		})
		ui.Column(c).Key("ghost-pbody").Grow(1).MinHeight(0).Clip().Padding(4, 10, 0, 10).Children(func() {
			for _, ln := range g.term {
				ui.Text(c, ln).SingleLine().Font("monospace").FontSize(termFont).TextColor(t.Tx2)
			}
		})
	})
}

// ghostTermLines 取当前终端画面的最后 n 行非空文字(没有终端 / 还没有画面返回 nil)。
func (a *App) ghostTermLines(n int) []string {
	if len(a.ide.tps) == 0 {
		return nil
	}
	tp := a.ide.tpc()
	if tp == nil || tp.scr == nil {
		return nil
	}
	var out []string
	for _, ln := range tp.scr.Lines[min(len(tp.scr.Lines), max(0, tp.scr.ScreenTop)):] {
		var sb strings.Builder
		for _, r := range ln {
			sb.WriteString(r.Text)
		}
		if s := strings.TrimRight(sb.String(), " "); s != "" {
			out = append(out, s)
		}
	}
	if len(out) > n {
		out = out[len(out)-n:]
	}
	return out
}
