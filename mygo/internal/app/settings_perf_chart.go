package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 性能页的趋势折线图(设计稿 settings.js「趋势」:悬停显示数值与十字线,硬要求):
// 鼠标在曲线上移动时出现竖向十字线 + 当前点的圆点 + 跟随鼠标的悬浮卡(时间与指标值),移出即全部消失。
//
// @author ixxxxoooo

const perfChartH = 120 // 曲线区高度(DIP)

// perfLineChart 画一张趋势图。
// 参数 title 是指标名("RSS" / "Go 堆" / "CPU"),vals 是各采样点的值,labels 是对应的时间标签,col 是曲线色。
func perfLineChart(a *App, c *ui.Context, title string, vals []float64, labels []string, col ui.Color) {
	t := a.tk
	if a.set.perfHover == nil {
		a.set.perfHover = map[string]int{}
	}
	n := len(vals)
	a.box(c).Padding(10, 12).Gap(6).Children(func() {
		ui.Text(c, title).FontWeight(500)
		cur := -1 // 本帧鼠标所在的采样点;-1 = 没有悬停
		plot := ui.Box(c).Grow(1).Height(perfChartH).Clip()
		plotW := plot.Bounds().W // 上一帧的宽度,用来决定悬浮卡放左边还是右边
		plot.Children(func() {
			ui.Box(c).Grow(1).Draw(func(p *ui.Painter, r ui.Rect) {
				drawPerfSeries(p, r, vals, col, cur, t.Raised)
			})
			// 一排等宽的透明感应条,一条对应一个采样点;不挂原生 Tooltip,悬浮信息统一由下面的悬浮卡显示。
			ui.Row(c).Absolute().Left(0).Right(0).Top(0).Bottom(0).AlignItems(ui.Stretch).Children(func() { // 必须拉满高度,否则感应条高为 0、永远悬停不到
				for i := 0; i < n; i++ {
					strip := ui.Box(c.Key(fmt.Sprintf("perf-%s-%d", title, i))).Grow(1).Basis(0)
					if strip.Hovered() {
						cur = i
					}
				}
			})
			if cur >= 0 && cur < len(labels) {
				a.perfHoverCard(c, title, labels[cur], vals[cur], col, float32(cur)/float32(max(n-1, 1)), plotW)
			}
		})
		if prev, ok := a.set.perfHover[title]; !ok || prev != cur {
			a.set.perfHover[title] = cur
			a.frameSoon() // 悬停点变了:下一帧让十字线与悬浮卡跟上
		}
		if n > 0 && n <= len(labels) {
			ui.Text(c, "最新 "+labels[n-1]+" · "+fmtVal(title, vals[n-1])).FontSize(theme.FsCap).TextColor(t.Tx3).FontFeatures("tnum")
		}
	})
}

// perfHoverCard 画跟随鼠标的悬浮卡:第一行时间,第二行「色点 + 指标值」。
// 参数 title 是指标名,label 是该点的时间,v 是该点的值,col 是曲线色,
// frac 是该点在横轴上的位置(0 – 1),plotW 是曲线区宽度:点靠近右侧时卡片翻到十字线左边,免得被裁掉。
func (a *App) perfHoverCard(c *ui.Context, title, label string, v float64, col ui.Color, frac, plotW float32) {
	t := a.tk
	card := ui.Column(c.Key("perf-card-"+title)).Absolute().Top(6).Gap(2).Padding(5, 9).
		Background(t.Raised).Radius(t.RSm()).Border(1, t.Bd2).PassThrough()
	for _, s := range t.Shadow2() {
		card.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	if x := frac * plotW; frac > .6 {
		card.Right(plotW - x + 10)
	} else {
		card.Left(x + 10)
	}
	card.Children(func() {
		ui.Text(c, label).FontSize(theme.FsCap).TextColor(t.Tx3).FontFeatures("tnum")
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Width(8).Height(8).Background(col).Radius(4)
			ui.Text(c, fmtVal(title, v)).FontWeight(600).FontFeatures("tnum")
		})
	})
}

// fmtVal 把指标值格式化成「名称 数值」,CPU 用百分比,其余按字节。
func fmtVal(title string, v float64) string {
	if title == "CPU" {
		return title + " " + fmt.Sprintf("%.1f%%", v)
	}
	return title + " " + diag.FormatBytes(uint64(v))
}

// drawPerfSeries 画曲线;hover >= 0 时再画竖向十字线和该点的圆点(ring 是圆点外圈色)。
func drawPerfSeries(p *ui.Painter, r ui.Rect, vals []float64, col ui.Color, hover int, ring ui.Color) {
	if len(vals) < 2 {
		return
	}
	lo, hi := vals[0], vals[0]
	for _, v := range vals {
		lo, hi = min(lo, v), max(hi, v)
	}
	if hi == lo {
		hi = lo + 1
	}
	pts := make([][2]float32, len(vals))
	for i, v := range vals {
		x := r.X + float32(i)/float32(len(vals)-1)*r.W
		y := r.Y + r.H - 2 - float32((v-lo)/(hi-lo))*(r.H-4)
		pts[i] = [2]float32{x, y}
	}
	p.StrokePolyline(pts, 1.5, col)
	if hover >= 0 && hover < len(pts) {
		x, y := pts[hover][0], pts[hover][1]
		p.StrokePolyline([][2]float32{{x, r.Y}, {x, r.Y + r.H}}, 1, col.Alpha(.5))
		p.Fill(ui.Rect{X: x - 5, Y: y - 5, W: 10, H: 10}, ring, 5)
		p.Fill(ui.Rect{X: x - 3.5, Y: y - 3.5, W: 7, H: 7}, col, 3.5)
	}
}
