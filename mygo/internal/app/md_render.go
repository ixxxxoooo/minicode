package app

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/mdcompat"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// mdHTMLBlock 渲染 Markdown 中的 HTML 块:已知的 <details> 折叠显示,其余提取可见文本或原样展示,不静默丢弃。
func (a *App) mdHTMLBlock(c *ui.Context, raw string) {
	raw = strings.TrimRight(raw, "\n")
	t := a.tk
	if sum, body, ok := mdcompat.ParseDetails(raw); ok {
		title := sum
		if title == "" {
			title = "详情"
		}
		ui.Column(c).Children(func() {
			ui.Text(c, title).FontWeight(650).FontSize(theme.FsCap).TextColor(t.Tx2).Margin(0, 0, 4, 0)
			a.foldBody(c, foldKey("html-d", mdShortHash(raw)), body, len(body) > 480, false, func() {
				a.copyOut(body, "正文")
			})
		})
		return
	}
	if vis := mdcompat.VisibleTextFromHTML(raw); vis != "" && vis != strings.TrimSpace(raw) {
		ui.Text(c, vis).TextColor(t.Tx1).FontSize(a.FlowCapFont())
		return
	}
	key := foldKey("html", mdShortHash(raw))
	if len(raw) > 600 {
		a.foldBody(c, key, raw, true, false, func() { a.copyOut(raw, "HTML") })
		return
	}
	// 复制按钮只有一个图标,浮在右上角,不单独占一行;正文右侧留出按钮的位置,免得第一行被盖住。
	ui.Box(c).Background(t.Sunken).Radius(t.RSm()).Border(1, t.Bd1).Padding(8, 32, 8, 10).Children(func() {
		ui.ScrollHorizontal(c).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, raw).Font(theme.CodeFont()).FontSize(theme.FsCap).TextColor(t.Tx2).NoWrap()
		})
		kit.Ib(c, "copy", kit.IbOpt{Small: true, Tip: "复制 HTML"}).Absolute().Top(3).Right(3).OnClick(func() {
			a.copyOut(raw, "HTML")
		})
	})
}

func mdShortHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8])
}
