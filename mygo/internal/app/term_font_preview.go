package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// @author ixxxxoooo

// termFontPreview 画终端字体预览:用终端真正会用的字体链渲染各类字符,右侧给每类的字形覆盖结论。
// 参数:pref 要预览的字体(可以是逗号分隔的回退链);applied 表示它就是当前生效的字体。
func (a *App) termFontPreview(c *ui.Context, pref string, applied bool) {
	chainStr := buildTermFontChain(pref, a.prefs.Bool("term.nerdFallback", true))
	a.fontPreview(c, pref, applied, chainStr, float32(a.prefs.Int(pTermFont, 12)), "跟随编辑器字体")
}

// codeFontPreview 画代码字体预览:用编辑器真正会用的字体链(加连字设置)渲染各类字符。
func (a *App) codeFontPreview(c *ui.Context, pref string, applied bool) {
	use := pref
	if firstFamily(use) == "" || firstFamily(use) == "monospace" {
		use = theme.DefaultCodeFamilies
	}
	chainStr := buildTermFontChain(use, false)
	a.fontPreview(c, pref, applied, chainStr, float32(a.prefs.Int(pEdFont, edFontDefault)), "默认字体链")
}

// fontPreview 是两种预览共用的画法。chainStr 是用来渲染与检查字形的完整字体链,fs 是预览字号,emptyTitle 是空值时的标题。
func (a *App) fontPreview(c *ui.Context, pref string, applied bool, chainStr string, fs float32, emptyTitle string) {
	t := a.tk
	fam := firstFamily(pref)
	chain := parseFontFamilies(chainStr)
	title := "预览"
	switch {
	case fam == "":
		title += " · " + emptyTitle
	case fam == "monospace":
		title += " · 系统等宽"
	default:
		title += " · " + fam
	}
	_, known, installed := fontInstalledLabel(fam)
	ui.Row(c).Gap(8).AlignItems(ui.Center).Margin(4, 0, 0, 0).Children(func() {
		ui.Text(c, title).FontSize(theme.FsCap).FontWeight(500).TextColor(t.Tx2).SingleLine().Shrink(1).MinWidth(0)
		if applied {
			kit.Tag(c, "当前使用", kit.KAc)
		}
		ui.Spacer(c).Grow(1)
	})
	if known && !installed && fam != "" {
		ui.Text(c, "这个字体还没安装,下面是回退字体的效果。").FontSize(theme.FsCap).TextColor(t.Warn)
	}
	var nerdMissing bool
	ui.Column(c).Padding(4, 10).Radius(t.RSm()).Background(t.BG).Border(1, t.Bd1).Children(func() {
		for i, cat := range glyphCats {
			res := chainCoverage(chain, cat)
			if cat.nerd && res.level == covMissing {
				nerdMissing = true
			}
			r := ui.Row(c).MinHeight(theme.Px(26)).Gap(10).AlignItems(ui.Center)
			if i > 0 {
				r.BorderWidth(1, 0, 0, 0).BorderColor(t.Bd1)
			}
			r.Children(func() {
				ui.Text(c, cat.name).FontSize(theme.FsCap).TextColor(t.Tx3).Width(theme.Px(78)).Shrink(0).SingleLine()
				ui.Text(c, cat.sample).Font(chainStr).FontSize(fs).TextColor(t.Tx1).NoWrap().Grow(1).MinWidth(0)
				covChip(c, t, res)
			})
		}
	})
	if nerdMissing {
		a.nerdInstallHint(c)
	}
}

// covChip 是一类字符右侧的覆盖结论小标。
func covChip(c *ui.Context, t *theme.Tokens, r catResult) {
	switch r.level {
	case covOwn:
		ui.Row(c).Gap(3).AlignItems(ui.Center).Shrink(0).Children(func() {
			kit.Ic(c, "check", kit.S12).TextColor(t.Ok)
			ui.Text(c, "自带").FontSize(theme.FsCap).TextColor(t.Ok)
		})
	case covFallback:
		tip := "这个字体没有这些字形,由回退字体补上(显示正常)"
		if r.by != "" {
			tip = "这个字体没有这些字形,由「" + r.by + "」补上(显示正常)"
		}
		ui.Text(c, "回退补全").FontSize(theme.FsCap).TextColor(t.Tx3).Tooltip(tip).Shrink(0)
	case covMissing:
		kit.Tag(c, "缺字形 "+itoa(r.miss), kit.KErr).Tooltip("整条字体链里都没有这些字符,终端会显示成方框")
	}
}

// nerdInstallHint 在预览下方提示 Nerd 图标缺字形,并给「一键安装」(装完预览与终端同时生效)。
func (a *App) nerdInstallHint(c *ui.Context) {
	t := a.tk
	const fam = "JetBrainsMono Nerd Font"
	st := a.set.fontInst[fam]
	ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(6, 10).Radius(t.RSm()).Background(t.WarnSoft).Children(func() {
		ui.Text(c, "Powerline / Nerd 图标会显示成方框:需要一个 Nerd Font。").FontSize(theme.FsCap).TextColor(t.Warn).Grow(1).MinWidth(0)
		if ok, known := fontInstalled(fam); known && ok {
			kit.Btn(c, "选用 "+fam, kit.BtnOpt{Small: true, Icon: "check"}).OnClick(func() { a.termFontPick(fam) })
			return
		}
		a.fontInstallControl(c, fam, st)
	})
}
