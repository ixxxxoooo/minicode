package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// mdTheme 把设计稿的 .md 样式(13px / 1.65、h3 14px、行内代码 11.5px)映射成 markdown.Theme。
func (a *App) mdTheme() *markdown.Theme {
	t := a.tk
	th := markdown.DefaultTheme(t.UITheme())
	th.FontSize = theme.FsRead * a.ChatFontScale()
	th.LineHeight = 1.65
	th.Text, th.Strong, th.Muted = t.Tx1, t.Tx1, t.Tx3
	th.Link = t.AcTx
	th.CodeSize = 11.5 / theme.FsRead
	th.CodeBackground = t.Fill2
	th.Code = t.Tx1
	th.BlockSpacing = .46
	th.ItemSpacing = .08
	th.ListIndent = 1.4
	th.Headings[2] = markdown.HeadingStyle{Size: 14.0 / theme.FsRead, Weight: 650, Top: .6, Bottom: .3}
	th.CodeBlockBackground = t.Sunken
	th.CodeBlockBorder = t.Bd1
	th.CodeBlockRadius = t.RMd()
	th.QuoteBar = t.Bd3
	th.QuoteText = t.Tx2
	th.TableBorder = t.Bd1
	return th
}

// mdOptions 返回(并缓存)会话流用的 Markdown 选项;换主题时清掉。
func (a *App) mdOptions() *markdown.Options {
	th := a.mdTheme()
	if a.md == nil || a.mdThemeKey != a.pal.ID+"|"+itoa(a.prefs.Int(pFontScale, 100)) {
		a.mdThemeKey = a.pal.ID + "|" + itoa(a.prefs.Int(pFontScale, 100))
		a.md = &markdown.Options{Theme: th, OnLink: a.openMdLink, LinkMenu: a.linkMenu}
		a.hookMDFold(a.md)
	}
	return a.md
}

var _ = ui.Transparent
