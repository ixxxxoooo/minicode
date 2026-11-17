package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

const (
	pTermFontManual   = "term.font"
	termFontTabCommon = 0
	termFontTabAll    = 1
)

func (a *App) termFontManualBuf() *string {
	if a.set.bufs == nil {
		a.set.bufs = map[string]*string{}
	}
	buf, ok := a.set.bufs[pTermFontManual]
	if !ok {
		s := a.prefs.Str(pTermFontManual, "")
		buf = &s
		a.set.bufs[pTermFontManual] = buf
	}
	return buf
}

// termFontPickerBody 设置 → 终端里的字体选择器(常见列表 + 全量搜索 + 手动输入 + 预览)。
func (a *App) termFontPickerBody(c *ui.Context) {
	a.startFontScan()
	t := a.tk
	tab := a.set.termFontTab
	if tab != termFontTabCommon && tab != termFontTabAll {
		tab = termFontTabCommon
	}
	cur := a.prefs.Str(pTermFontManual, "")
	buf := a.termFontManualBuf()
	ui.Column(c).Padding(8, 12, 10, 12).Gap(8).Children(func() {
		if i := kit.Seg(c, []kit.SegItem{{Text: "常见等宽"}, {Text: "全部字体"}}, tab, kit.SegOpt{Key: "tfont-tab"}); i >= 0 && i != tab {
			a.set.termFontTab = i
		}
		if tab == termFontTabAll {
			ui.Row(c).Height(theme.Px(28)).PaddingX(8).Gap(6).Radius(t.RSm()).Background(t.Raised).Border(1, t.Bd2).AlignItems(ui.Center).Children(func() {
				kit.Ic(c, "search", kit.S12).TextColor(t.Tx3)
				ui.TextInputBase(c.Key("tfont-q"), &a.set.termFontQ).Grow(1).FontSize(theme.FsCap + .5).Placeholder("搜索字体…").PlaceholderColor(t.Tx4)
			})
		}
		kit.VScroll(c).Key("tfont-list").MaxHeight(220).Children(func() {
			if tab == termFontTabCommon {
				a.termFontCommonList(c, cur)
			} else {
				a.termFontAllList(c, cur, strings.ToLower(strings.TrimSpace(a.set.termFontQ)))
			}
		})
		ui.Text(c, "手动输入(支持逗号分隔回退链)").FontSize(theme.FsCap).TextColor(t.Tx3)
		ui.Row(c).Height(theme.Px(28)).PaddingX(8).Radius(t.RSm()).Background(t.Raised).Border(1, t.Bd2).Grow(1).Children(func() {
			ui.TextInputBase(c.Key("tfont-manual"), buf).Grow(1).Font("monospace").FontSize(theme.FsCap + .5).
				Placeholder(`如 "JetBrains Mono", Menlo, monospace`).PlaceholderColor(t.Tx4)
		})
		ui.Row(c).Gap(8).Children(func() {
			kit.Btn(c, "应用字体", kit.BtnOpt{Small: true, Icon: "check"}).OnClick(func() {
				a.prefs.Set(pTermFontManual, strings.TrimSpace(*buf))
				a.applyBehavior()
				a.flash("终端字体已更新")
			})
			if a.prefs.Has(pTermFontManual) {
				kit.Btn(c, "清空", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
					a.prefs.Reset(pTermFontManual)
					*buf = ""
					a.applyBehavior()
				})
			}
		})
		ui.Text(c, "预览").FontSize(theme.FsCap).TextColor(t.Tx3)
		previewFam := buildTermFontChain(strings.TrimSpace(*buf), a.prefs.Bool("term.nerdFallback", true))
		fs := float32(a.prefs.Int(pTermFont, 12)) - .5
		ui.Box(c).Padding(8, 10).Radius(t.RSm()).Background(t.BG).Border(1, t.Bd1).Children(func() {
			ui.Text(c, termFontPreview).Font(previewFam).FontSize(fs).TextColor(t.Tx1).NoWrap()
		})
	})
}

func (a *App) termFontPick(family string) {
	a.prefs.Set(pTermFontManual, family)
	*a.termFontManualBuf() = family
	a.applyBehavior()
	a.flash("终端字体已更新")
}

func (a *App) termFontCommonList(c *ui.Context, cur string) {
	t := a.tk
	for i, fam := range commonMonoFonts {
		label := fam
		if label == "" {
			label = "系统等宽"
		}
		tag, known, ok := fontInstalledLabel(fam)
		disabled := known && !ok && fam != ""
		row := rowBox(a, c, i == 0, 40)
		if cur == fam {
			row.Background(t.AcSoft)
		}
		if disabled {
			row.Opacity(.55)
		}
		row.Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
				ui.Text(c, label).FontWeight(500)
				ui.Text(c, tag).FontSize(theme.FsCap).TextColor(map[bool]ui.Color{true: t.Tx3, false: t.Warn}[ok || fam == ""])
			})
			if !disabled {
				kit.Btn(c, "选用", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
					a.termFontPick(fam)
				})
			}
		})
	}
}

func (a *App) termFontAllList(c *ui.Context, cur, q string) {
	t := a.tk
	fams := systemMonoFamilies()
	if len(fams) == 0 {
		ui.Text(c, "正在后台枚举系统字体…").FontSize(theme.FsCap).TextColor(t.Tx3).Padding(8, 4)
		return
	}
	n := 0
	for _, f := range fams {
		if q != "" && !strings.Contains(strings.ToLower(f.Name), q) {
			continue
		}
		fam := f.Name
		row := rowBox(a, c, n == 0, 36)
		n++
		if cur == fam {
			row.Background(t.AcSoft)
		}
		pick := fam
		row.Children(func() {
			ui.Text(c, fam).Grow(1).MinWidth(0).FontWeight(500)
			kit.Btn(c, "选用", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
				a.termFontPick(pick)
			})
		})
	}
	if n == 0 {
		ui.Text(c, "没有匹配的等宽字体").FontSize(theme.FsCap).TextColor(t.Tx3).Padding(8, 4)
	}
}

// termFontPickerBrief 终端弹层里的精简字体行。
func (a *App) termFontPickerBrief(c *ui.Context) {
	a.startFontScan()
	cur := a.prefs.Str(pTermFontManual, "")
	ui.Text(c, "字体").FontSize(theme.FsCap).TextColor(a.tk.Tx3).Margin(6, 0, 2, 4)
	for _, fam := range commonMonoFonts {
		if fam == "" || fam == "Courier New" {
			continue
		}
		label := fam
		_, known, ok := fontInstalledLabel(fam)
		dis := known && !ok
		if dis {
			a.mi(c, "tf-"+fam, "", label+"(未安装)", "", false)
			continue
		}
		if a.mi(c, "tf-"+fam, "", label, "", cur == fam).Clicked() {
			a.termFontPick(fam)
		}
	}
	if a.mi(c, "tf-more", "gear", "打开字体选择器…", "", false).Clicked() {
		a.popup = ""
		a.openSettings("terminal")
	}
}
