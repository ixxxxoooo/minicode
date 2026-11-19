package app

import (
	"fmt"

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
	th.CodeFont = theme.CodeFont() // 行内代码与代码块用同一条代码字体链(设置 → 编辑器 → 字体)
	th.CodeSize = 12 / theme.FsRead
	// 行内代码跟随主题强调色(设计稿 .md code):字色 AcTx,底色强调色 11%。
	th.Code, th.CodeBackground = a.inlineCodeColors()
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
	th.TableHeader = t.Fill1                                                  // 设计稿 .md th:表头底 fill-1
	th.TableCellPadding = [2]float32{10.0 / theme.FsRead, 5.0 / theme.FsRead} // 设计稿 .md td:5px 10px
	return th
}

// inlineCodeColors 返回行内代码的字色与底色:中性(正文色 + 8.5% 墨色底),参照 BaoCode,
// 不再满屏强调色(强调色留给链接、选中与可点的路径)。助手正文(Markdown)与用户消息共用,保证两处一致。
func (a *App) inlineCodeColors() (fg, bg ui.Color) {
	t := a.tk
	return t.Tx1, t.Fill2
}

// mdOptions 返回(并缓存)会话流用的 Markdown 选项;换主题时清掉。
func (a *App) mdOptions() *markdown.Options {
	th := a.mdTheme()
	// 缓存键含强调色:行内代码与链接都取强调色,只改强调色(主题不变)也要重建。
	key := a.pal.ID + "|" + itoa(a.prefs.Int(pFontScale, 100)) + "|" + fmt.Sprint(a.tk.Ac) + "|" + theme.CodeFont()
	if a.md == nil || a.mdThemeKey != key {
		a.mdThemeKey = key
		opts := &markdown.Options{Theme: th, OnLink: a.openMdLink, LinkMenu: a.linkMenu, HTML: a.mdHTMLBlock,
			CodeLink: a.codeLink, LinkIcon: a.linkIcon} // 行内代码写的本机路径可点开,前面带文件 / 文件夹图标(path_ref.go)
		if a.proj != "" {
			opts.LoadImage = markdown.FileImages(a.proj)
		} else if a.ide.fs != nil {
			opts.LoadImage = markdown.FileImages(a.ide.fs.Abs(""))
		}
		a.md = opts
		a.hookMDFold(a.md)
	}
	return a.md
}

// planMdOptions 计划小卡概要的 Markdown 选项:在会话流样式基础上缩到说明字号、正文用次要色、行距收紧。
func (a *App) planMdOptions() *markdown.Options {
	key := a.mdThemeKey + "|plan|" + fmt.Sprint(a.FlowCapFont())
	if a.planMd == nil || a.planMdKey != key || a.md == nil {
		base := a.mdOptions()
		th := a.mdTheme()
		th.FontSize = a.FlowCapFont()
		th.LineHeight = 1.5
		th.Text = a.tk.Tx2
		th.BlockSpacing = .3
		th.CodeSize = 11 / theme.FsCap
		// Options 含 sync.Map,不能整体复制:新建并逐项带上基础选项的回调 / 图片加载。
		a.planMd = &markdown.Options{Theme: th, OnLink: base.OnLink, LinkMenu: base.LinkMenu, LoadImage: base.LoadImage}
		a.planMdKey = key
	}
	return a.planMd
}

// thinkMdOptions 思考正文的 Markdown 选项:说明字号、三级灰(思考是过程,不抢正文),行距收紧。
func (a *App) thinkMdOptions() *markdown.Options {
	key := a.mdThemeKey + "|think|" + fmt.Sprint(a.FlowCapFont())
	if a.thinkMd == nil || a.thinkMdKey != key || a.md == nil {
		base := a.mdOptions()
		th := a.mdTheme()
		th.FontSize = a.FlowCapFont()
		th.LineHeight = 1.6
		th.Text = a.tk.Tx3
		th.BlockSpacing = .4
		th.CodeSize = 11 / theme.FsCap
		a.thinkMd = &markdown.Options{Theme: th, OnLink: base.OnLink, LinkMenu: base.LinkMenu, LoadImage: base.LoadImage}
		a.thinkMdKey = key
	}
	return a.thinkMd
}

var _ = ui.Transparent
