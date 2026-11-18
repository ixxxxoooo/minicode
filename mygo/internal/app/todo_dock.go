// 待办坞:会话里有待办时固定在输入框上方的一行,点开看完整清单(设计稿 .todock)。
//
// @author ixxxxoooo
package app

import (
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// todoDockMaxH 是展开后清单区的最大高度,超出在内部滚动。
const todoDockMaxH = 180

// todoDockSummary 汇总一份待办:已完成条数,以及标题行要显示的「当前这一步」。
// 参数 td:待办清单。返回 done(已完成条数)、now(当前文案)、active(now 是否为进行中的那一条)。
func todoDockSummary(td *core.Todos) (done int, now string, active bool) {
	next := ""
	for _, it := range td.Items {
		switch it.Status {
		case core.TodoCompleted:
			done++
		case core.TodoInProgress:
			if !active {
				active, now = true, it.ActiveForm
				if now == "" {
					now = it.Content
				}
			}
		default:
			if next == "" {
				next = it.Content
			}
		}
	}
	switch {
	case active:
	case done == len(td.Items):
		now = "全部完成"
	default:
		now = "下一步:" + next
	}
	return done, now, active
}

// todoRing 画 14px 的进度环:一圈底轨 + 从 12 点方向顺时针走的进度弧。
// 参数 frac:完成比例 0..1;track / col:底轨与进度弧的颜色。返回环元素。
func todoRing(c *ui.Context, frac float32, track, col ui.Color) ui.Element {
	frac = max(0, min(1, frac))
	e := ui.Box(c).Size(14, 14).Shrink(0)
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		arc := func(sweep float64, color ui.Color) {
			const n = 40
			pts := make([][2]float32, 0, n+1)
			for i := 0; i <= n; i++ {
				ang := -math.Pi/2 + sweep*float64(i)/n
				pts = append(pts, [2]float32{cx + 5*float32(math.Cos(ang)), cy + 5*float32(math.Sin(ang))})
			}
			p.StrokePolyline(pts, 1.8, color)
		}
		arc(2*math.Pi, track)
		if frac > 0 {
			arc(2*math.Pi*float64(frac), col)
		}
	})
	return e
}

// todoDock 画待办坞:取时间线里最新的一份待办,固定在输入框上方(任务坞之下)。
// 没有待办、或全部完成且回合已结束时不画。参数 c:界面上下文。无返回值。
func (a *App) todoDock(c *ui.Context) {
	td := a.latestTodos()
	if td == nil || len(td.Items) == 0 {
		return
	}
	total := len(td.Items)
	done, now, active := todoDockSummary(td)
	running := a.st.Phase == core.PhaseRunning
	if done == total && !running {
		return
	}
	t := a.tk
	open := a.todoDockOpen
	box := ui.Column(c).Key("todo-dock").Margin(0, 0, 6, 0).Radius(t.RMd()).Background(t.Raised).Border(1, t.Bd2).Clip()
	kit.FloatLift(box) // 流光:浮在会话流上的玻璃
	box.Children(func() {
		head := ui.ButtonBase(c).Key("todo-dock-head").Height(theme.Px(30)).PaddingX(10).Gap(8).Justify(ui.Start).AlignItems(ui.Center).
			Cursor(ui.CursorPointer).Tooltip(map[bool]string{true: "收起待办", false: "展开待办"}[open])
		if head.Hovered() {
			head.Background(t.Fill1)
		}
		if head.Clicked() {
			a.todoDockOpen = !open
			log.Info("待办坞", "展开", !open, "done", done, "total", total)
		}
		head.Children(func() {
			todoRing(c, float32(done)/float32(total), t.Fill3, t.Ac)
			ui.Text(c, "待办").FontWeight(500).TextColor(t.Tx1).Shrink(0)
			ui.Text(c, itoa(done)+" / "+itoa(total)).TextColor(t.Tx3).FontFeatures("tnum").Shrink(0)
			cur := ui.Text(c, now).Grow(1).MinWidth(0).SingleLine().TextColor(t.Tx2)
			if active && running {
				kit.RunText(cur) // 进行中的那一步:流光 / 呼吸(随设置)
			}
			icon := "chevR"
			if open {
				icon = "chevD"
			}
			kit.Ic(c, icon, kit.S12).TextColor(t.Tx3)
		})
		if !open {
			return
		}
		ui.Box(c).Height(1).Background(t.Bd1)
		kit.VScroll(c).Key("todo-dock-body").MaxHeight(todoDockMaxH).Children(func() {
			ui.Column(c).Padding(6, 10, 8, 10).Children(func() { a.todoBody(c, td) })
		})
	})
}
