package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// 「回到最新」(设计稿 flow-queue):滚离底部超过 32px 才出现,输入框正上方水平居中的圆形向下箭头;
// 离开期间有新内容时右上角亮一个强调色小圆点。子 Agent 过程流(taskSt)同形态,居中放在各自滚动区底部。

func (a *App) flowFollowBottom() bool {
	return !a.flowNoFollow && a.flowList.AtEnd()
}

// flowTailMark 取会话末尾的「变化标记」:条目数 × 8 + 末条助手正文长度的粗粒度,流式输出时也会变。
func flowTailMark(items []core.Item) int {
	n := len(items) * 8
	if len(items) > 0 {
		if as, ok := items[len(items)-1].(*core.Assistant); ok {
			n += len(as.Text) / 64
		}
	}
	return n
}

// toLatestBtn 画圆形「回到最新」按钮,居中、贴着所在滚动区的底边;fresh = 带「有新内容」圆点。
func (a *App) toLatestBtn(c *ui.Context, key string, fresh bool, bottom float32, onClick func()) {
	t := a.tk
	ui.Row(c).Key(key + "-wrap").Absolute().Left(0).Right(0).Bottom(bottom).Justify(ui.Center).ZIndex(5).PassThrough().Children(func() { // 整行容器不吃点击,只有圆钮本身响应
		b := ui.ButtonBase(c).Key(key).Size(30, 30).Radius(15).Justify(ui.Center).AlignItems(ui.Center).
			Background(t.Raised).Border(1, t.Bd2).Cursor(ui.CursorPointer).Transition(kit.Fade).
			Tooltip("回到最新").Label("回到最新")
		for _, s := range t.Shadow2() {
			b.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
		}
		col := t.Tx2
		if b.Hovered() {
			// Fill2 是半透明的墨色;直接当底色会让按钮变透明、露出后面滚动的文字,要先叠在不透明的 Raised 上。
			b.Background(t.Fill2.Over(t.Raised))
			col = t.Tx1
		}
		b.OnClick(onClick)
		b.Children(func() {
			kit.Ic(c, "arrowDown", kit.S12).TextColor(col)
			if fresh {
				ui.Box(c).Absolute().Top(3).Right(3).Size(7, 7).Radius(3.5).Background(t.Ac).Border(2, t.Raised)
			}
		})
	})
}

func (a *App) flowScrollBtn(c *ui.Context) {
	if a.flowFollowBottom() || a.flowList.AtEnd() || a.flowSt.MaxY-a.flowSt.Y < 32 || len(a.st.Items) == 0 {
		return
	}
	fresh := flowTailMark(a.st.Items) != a.flowAwayLen
	a.toLatestBtn(c, "to-latest", fresh, a.dockH+10, func() {
		a.flowNoFollow = false
		a.flowAwayLen = flowTailMark(a.st.Items)
		a.flowList.ScrollToEnd()
	})
}

func (a *App) taskScrollBtn(c *ui.Context) {
	if a.taskNoFollow || a.taskSt.MaxY-a.taskSt.Y < 32 {
		return
	}
	a.toLatestBtn(c, "to-latest-task", false, 10, func() { a.taskNoFollow = false; a.taskSt.Y = 1e9 })
}
