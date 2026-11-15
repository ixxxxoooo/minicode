package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// toggleBlockComment 块注释 ⌥⇧A:有块语法则包裹,否则改用行注释并提示(设计稿 ide-nav)。
func (a *App) toggleBlockComment() {
	tb := a.ide.cur()
	if ok, why := a.tbCodeEditable(tb); !ok {
		a.toast = why
		return
	}
	st := tb.st
	if st.Run(editor.ToggleBlockComment) {
		return
	}
	lang := "该语言"
	if l := tb.buf.Language(); l != nil && l.Name != "" {
		lang = l.Name
	}
	if !st.Run(editor.ToggleComment) {
		a.toast = lang + " 没有块注释"
		return
	}
	a.showBlockCmtHint(lang + " 没有块注释,已改用行注释")
}

func (a *App) showBlockCmtHint(msg string) {
	a.ux.blockCmtHint = msg
	a.ux.blockCmtUntil = a.now().Add(5 * time.Second)
	uxLog.Info("块注释退化为行注释", "msg", msg)
}

func (a *App) blockCmtBanner(c *ui.Context, tb *ideTab) {
	if a.ux.blockCmtHint == "" {
		return
	}
	if a.now().After(a.ux.blockCmtUntil) {
		a.ux.blockCmtHint = ""
		return
	}
	t := a.tk
	ui.Row(c).Key("block-cmt-hint").Height(32).Shrink(0).PaddingX(12).Gap(8).AlignItems(ui.Center).
		Background(t.InfoSoft).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
		kit.Ic(c, "info", kit.S12).TextColor(t.Info)
		ui.Text(c, a.ux.blockCmtHint).Grow(1).SingleLine().TextColor(t.Tx2)
		if kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "关闭"}).Clicked() {
			a.ux.blockCmtHint = ""
		}
	})
	_ = tb
	_ = theme.FsBody
}
