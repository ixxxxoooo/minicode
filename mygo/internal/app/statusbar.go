package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 状态栏的一个小项(.sbar .it:18 高、悬停 fill-1)。
func (a *App) sbItem(c *ui.Context, key string, children func()) ui.Element {
	t := a.tk
	b := ui.ButtonBase(c).Key("sb-" + key).Height(18).PaddingX(5).Gap(4).Radius(t.RXs()).AlignItems(ui.Center).Cursor(ui.CursorPointer).Shrink(0).Transition(kit.Fade)
	if b.Hovered() {
		b.Background(t.Fill1).TextColor(t.Tx1)
	}
	return b.Children(children)
}

// statusbar 是状态栏:两种形态共用,IDE 专属项在 IDE 形态才出现。
func (a *App) statusbar(c *ui.Context) {
	t := a.tk
	h := float32(theme.HStatus)
	if t.Scheme.ID == "b" {
		h = 24
	}
	r := ui.Row(c).Height(h).PaddingX(8).Gap(6).FontSize(11).TextColor(t.Tx3).Shrink(0).AlignItems(ui.Center)
	if t.Scheme.ID != "b" {
		r.BorderWidth(1, 0, 0, 0).BorderColor(t.Bd2)
	}
	r.Children(func() {
		if a.branch != "" {
			a.sbItem(c, "branch", func() { kit.Ic(c, "branch", kit.S12); ui.Text(c, a.branch) }).OnClick(func() { a.OpenQuick("branch", "") })
		}
		a.sbItem(c, "sync", func() { kit.Ic(c, "refresh", kit.S12); ui.Text(c, "↑1") })
		a.sbItem(c, "diag", func() {
			kit.Ic(c, "errc", kit.S12)
			ui.Text(c, "0")
			kit.Ic(c, "alert", kit.S12)
			ui.Text(c, "1")
		})
		if a.mode == ModeIDE {
			if tab := a.ide.cur(); tab != nil && !tab.diff && tab.st != nil {
				p := tab.st.CursorPos()
				a.sbItem(c, "pos", func() { ui.Text(c, "行 "+itoa(p.Line+1)+",列 "+itoa(p.Col+1)).FontFeatures("tnum") })
				a.sbItem(c, "enc", func() { ui.Text(c, "UTF-8") })
				lang := "纯文本"
				if l := tab.buf.Language(); l != nil {
					lang = l.Name
				}
				a.sbItem(c, "lang", func() { ui.Text(c, lang) })
				// 语言服务器尚未接入:只在演示数据里展示设计稿的状态项,真实模式不假装有连接
				if srv := lspName(lang); srv != "" && a.demoExtra {
					a.sbItem(c, "lsp", func() { kit.Ic(c, "okc", kit.S12).TextColor(t.Ok); ui.Text(c, srv) })
				}
			}
		}
		ui.Spacer(c).Grow(1)
		mode := a.sbItem(c, "mode", func() { kit.Ic(c, modeIcon(a.st.Mode), kit.S12); ui.Text(c, a.st.Mode.Label()) })
		mode.OnClick(func() { a.togglePop("sbcfg") })
		a.cfgPopup(c, "sbcfg", mode, true)
		model := a.sbItem(c, "model", func() { ui.Text(c, a.modelName()) })
		model.OnClick(func() { a.togglePop("sbcfg") })
		ctx := a.sbItem(c, "ctx", func() {
			ui.Text(c, "上下文 "+itoa(int(a.st.Usage.ContextRatio()*100+.5))+"%").FontFeatures("tnum")
		})
		ctx.OnClick(func() { a.togglePop("ctx") })
		a.ctxPopup(c, ctx)
		if a.update != "" {
			upd := a.sbItem(c, "upd", func() {
				kit.Ic(c, "download", kit.S12).TextColor(t.Ac)
				ui.Text(c, a.update).TextColor(t.Ac)
			})
			upd.OnClick(func() { a.togglePop("upd") })
			a.updPopup(c, upd)
		}
		if a.devBadge != "" && t.Scheme.ID != "x" {
			kit.Tag(c, a.devBadge, kit.KAc)
		}
	})
}

// lspName 返回某语言默认使用的语言服务器名(没有则为空)。
func lspName(lang string) string {
	switch lang {
	case "Rust":
		return "rust-analyzer"
	case "Go":
		return "gopls"
	case "Python":
		return "Pyright"
	case "TypeScript", "JavaScript":
		return "typescript-language-server"
	}
	return ""
}
