package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// @author ixxxxoooo

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

// firstFamily 取字体回退链里的首选字体(如 `"JetBrains Mono", Menlo` → JetBrains Mono)。
func firstFamily(chain string) string {
	if fs := parseFontFamilies(chain); len(fs) > 0 {
		return fs[0]
	}
	return ""
}

// termFontPickerBody 是设置 → 终端里的字体选择器:列表(整行可点,点了立刻生效)+ 实时预览 + 手动回退链。
// 预览跟随「鼠标悬停的字体 > 点过的字体 > 当前生效的字体」,下面逐类给出字形覆盖结论(见 font_cov.go)。
func (a *App) termFontPickerBody(c *ui.Context) {
	a.startFontScan()
	t := a.tk
	tab := a.set.termFontTab
	if tab != termFontTabCommon && tab != termFontTabAll {
		tab = termFontTabCommon
	}
	cur := a.prefs.Str(pTermFontManual, "")
	curFam := firstFamily(cur)
	buf := a.termFontManualBuf()
	hover := ""
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
		kit.VScroll(c).Key("tfont-list").MaxHeight(240).Children(func() {
			if tab == termFontTabCommon {
				for _, fam := range commonMonoFonts {
					a.termFontRow(c, fam, curFam, &hover)
				}
				return
			}
			a.termFontAllList(c, curFam, strings.ToLower(strings.TrimSpace(a.set.termFontQ)), &hover)
		})
		// 手动输入:边输边预览,点「应用」才写入偏好
		if *buf != a.set.termFontSeen {
			a.set.termFontSeen = *buf
			a.set.termFontPrev = strings.TrimSpace(*buf)
		}
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
				kit.Btn(c, "恢复系统等宽", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
					a.prefs.Reset(pTermFontManual)
					*buf, a.set.termFontPrev = "", ""
					a.applyBehavior()
				})
			}
		})
		prev := hover
		if prev == "" {
			prev = a.set.termFontPrev
		}
		if prev == "" {
			prev = cur
		}
		a.termFontPreview(c, prev, firstFamily(prev) == curFam)
	})
}

// termFontPick 点选一个字体:立刻写入偏好并生效,预览随之切到它。
func (a *App) termFontPick(family string) {
	a.prefs.Set(pTermFontManual, family)
	*a.termFontManualBuf() = family
	a.set.termFontSeen, a.set.termFontPrev = family, family
	a.applyBehavior()
	a.flash("终端字体已更新")
}

// termFontRow 画一行字体:左侧整块可点(名字用它自己的字体渲染,下面一行是安装状态),右侧是安装操作。
// 已安装的字体点一下就应用;没装的只切换预览(看回退效果)。hover 收集本帧悬停的字体,给预览用。
func (a *App) termFontRow(c *ui.Context, fam, curFam string, hover *string) {
	t := a.tk
	label := fam
	if label == "" {
		label = "系统等宽"
	}
	status, known, ok := fontInstalledLabel(fam)
	on := familyNamesEqual(fam, curFam) || fam == "" && curFam == ""
	st := a.set.fontInst[fam]
	row := ui.Row(c).Key("tfr-" + fam).Gap(4).AlignItems(ui.Center).Shrink(0)
	row.Children(func() {
		btn := ui.ButtonBase(c).Key("tfb-" + fam).Grow(1).MinWidth(0).Height(theme.Px(40)).PaddingX(8).Gap(8).Radius(t.RSm()).
			AlignItems(ui.Center).Cursor(ui.CursorPointer).Transition(kit.Fade)
		switch {
		case on:
			btn.Background(t.Sel)
		case btn.Hovered():
			btn.Background(t.Fill2)
		}
		if btn.Hovered() {
			*hover = fam
			if fam == "" {
				*hover = "monospace"
			}
		}
		btn.OnClick(func() {
			a.set.termFontPrev = fam
			if ok || fam == "" {
				a.termFontPick(fam)
			}
		})
		btn.Children(func() {
			ui.Box(c).Size(14, 14).Shrink(0).Children(func() {
				if on {
					kit.Ic(c, "check", kit.S14).TextColor(t.Ac)
				}
			})
			ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
				name := ui.Text(c, label).FontWeight(500).SingleLine().TextColor(t.Tx1)
				if ok && fam != "" {
					name.Font(fam + ", monospace") // 名字用它自己的字体渲染:先看一眼字形
				}
				sub := status
				if looksNerdFont(fam) {
					sub += " · Nerd Font"
				}
				col := t.Tx3
				if known && !ok {
					col = t.Warn
				}
				ui.Text(c, sub).FontSize(theme.FsCap).TextColor(col).SingleLine()
			})
		})
		if known && !ok {
			a.fontInstallControl(c, fam, st)
		}
	})
}

// fontInstallControl 是未安装字体右侧的安装操作:安装 / 安装中 / 失败重试 / 需手动安装。
func (a *App) fontInstallControl(c *ui.Context, fam string, st *fontInstall) {
	t := a.tk
	cask, has := fontCaskFor(fam)
	switch {
	case st != nil && st.running:
		ui.Row(c).Gap(6).AlignItems(ui.Center).Shrink(0).Children(func() {
			kit.Spin(c, t.Tx3)
			ui.Text(c, "安装中…").FontSize(theme.FsCap).TextColor(t.Tx3)
		})
	case !has:
		ui.Text(c, "需手动安装").FontSize(theme.FsCap).TextColor(t.Tx4).Shrink(0)
	default:
		if st != nil && st.failed != "" {
			ui.Text(c, "安装失败").FontSize(theme.FsCap).TextColor(t.Err).Tooltip(st.failed).Shrink(0)
		}
		kit.Btn(c, map[bool]string{true: "重试", false: "安装"}[st != nil && st.failed != ""],
			kit.BtnOpt{Small: true, Icon: "download", Tip: "brew install --cask " + cask}).OnClick(func() { a.installFont(fam) })
	}
}

// termFontAllList 是「全部字体」:系统里标记为等宽的字体族,支持搜索。
func (a *App) termFontAllList(c *ui.Context, curFam, q string, hover *string) {
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
		n++
		a.termFontRow(c, f.Name, curFam, hover)
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
		if known && !ok {
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
