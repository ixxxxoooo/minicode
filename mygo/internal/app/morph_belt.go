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

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// beltRect 是一块面板在舞台里的横向位置与宽度(布局像素)。
type beltRect struct{ x, w float32 }

// beltGeom 是某个形态下对话栏与工作区的几何。
type beltGeom struct {
	chat, ws     beltRect
	chatOn, wsOn bool // 对话栏 / 工作区此刻占位(不占位时宽度为 0)
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
	a.belt0, a.belt0Ok = a.beltGeomNow(a.stageW, a.tk.Scheme.GapPane), true
}

// beltStep 算本帧的摆位。不在平移中(没有真实窗口且没冻结 / 动效关 / 没记到旧几何 / 已走完)返回 on = false。
func (a *App) beltStep() beltFrame {
	if (a.win == nil && morphFreezeP == 0) || (a.screen != ScreenWork && a.screen != ScreenWelcome) || !a.belt0Ok || a.stageW <= 0 || a.morphAt.IsZero() {
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
	return f
}

// wsGhost 画被挤出舞台的旧工作区。
// 同一份工作区内容(编辑器 / 终端等控件的句柄)一帧里只能绑定一个元素,不能再画第二份,
// 所以幽灵是轻量版:面板外壳 + 标签条(标签名,当前标签高亮)+ 空白主体,不接收输入(切换期间整窗有指针罩)。
func (a *App) wsGhost(c *ui.Context, r beltRect) {
	t := a.tk
	p := a.pane(c).Key("ws-ghost").ClipX().Absolute().Left(r.x).Top(0).Bottom(0).Width(r.w).Shrink(0).Transition(noMotion).PassThrough()
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
	})
}
