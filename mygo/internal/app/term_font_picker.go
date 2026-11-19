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

// fontPickState 是一个字体选择器的界面状态(终端、代码各一份)。
type fontPickState struct {
	tab  int    // 0 常见等宽 / 1 全部字体
	q    string // 「全部字体」的搜索词
	seen string // 手动输入框上一帧的内容(变了就把预览切到它)
	prev string // 点过的字体(预览用;没装的字体只预览、不应用)
}

// fontPick 描述一个字体选择器:写哪个偏好、叫什么、空值代表什么、预览怎么画。
type fontPick struct {
	key       string                                         // 偏好键(空串 = 默认)
	name      string                                         // 提示用名字,如「终端字体」
	emptyName string                                         // 空值那一行的名字,如「系统等宽」
	resetText string                                         // 「恢复」按钮文字
	placehold string                                         // 手动输入框占位
	st        *fontPickState                                 // 界面状态
	preview   func(c *ui.Context, pref string, applied bool) // 预览
}

// termPick 是终端字体选择器的配置。
func (a *App) termPick() *fontPick {
	return &fontPick{key: pTermFontManual, name: "终端字体", emptyName: "跟随编辑器字体", resetText: "跟随编辑器字体",
		placehold: `如 "JetBrains Mono", Menlo, monospace`, st: &a.set.fpTerm, preview: a.termFontPreview}
}

// codePick 是代码字体(编辑器 / 差异 / 会话代码块)选择器的配置。
func (a *App) codePick() *fontPick {
	return &fontPick{key: pEdFamily, name: "代码字体", emptyName: "默认字体链(JetBrains Mono → Fira Code → Menlo)", resetText: "恢复默认",
		placehold: `如 "JetBrains Mono", Fira Code, Menlo`, st: &a.set.fpCode, preview: a.codeFontPreview}
}

func (a *App) fontManualBuf(fp *fontPick) *string {
	if a.set.bufs == nil {
		a.set.bufs = map[string]*string{}
	}
	buf, ok := a.set.bufs[fp.key]
	if !ok {
		s := a.prefs.Str(fp.key, "")
		buf = &s
		a.set.bufs[fp.key] = buf
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

// termFontPickerBody 是设置 → 终端里的字体选择器。
func (a *App) termFontPickerBody(c *ui.Context) { a.fontPickerBody(c, a.termPick()) }

// codeFontPickerBody 是设置 → 编辑器里的字体选择器(同一套交互,写 editor.fontFamily)。
func (a *App) codeFontPickerBody(c *ui.Context) { a.fontPickerBody(c, a.codePick()) }

// fontPickerBody 是字体选择器:列表(整行可点,点了立刻生效)+ 实时预览 + 手动回退链。
// 预览跟随「鼠标悬停的字体 > 点过的字体 > 当前生效的字体」,下面逐类给出字形覆盖结论(见 font_cov.go)。
func (a *App) fontPickerBody(c *ui.Context, fp *fontPick) {
	a.startFontScan()
	t := a.tk
	s := fp.st
	if s.tab != termFontTabCommon && s.tab != termFontTabAll {
		s.tab = termFontTabCommon
	}
	cur := a.prefs.Str(fp.key, "")
	curFam := firstFamily(cur)
	buf := a.fontManualBuf(fp)
	hover := ""
	ui.Column(c).Padding(8, 12, 10, 12).Gap(8).Children(func() {
		if i := kit.Seg(c, []kit.SegItem{{Text: "常见等宽"}, {Text: "全部字体"}}, s.tab, kit.SegOpt{Key: fp.key + "-tab"}); i >= 0 && i != s.tab {
			s.tab = i
		}
		if s.tab == termFontTabAll {
			ui.Row(c).Height(theme.Px(28)).PaddingX(8).Gap(6).Radius(t.RSm()).Background(t.Raised).Border(1, t.Bd2).AlignItems(ui.Center).Children(func() {
				kit.Ic(c, "search", kit.S12).TextColor(t.Tx3)
				ui.TextInputBase(c.Key(fp.key+"-q"), &s.q).Grow(1).FontSize(theme.FsCap + .5).Placeholder("搜索字体…").PlaceholderColor(t.Tx4)
			})
		}
		kit.VScroll(c).Key(fp.key + "-list").MaxHeight(240).Children(func() {
			if s.tab == termFontTabCommon {
				for _, fam := range commonMonoFonts {
					a.fontRow(c, fp, fam, curFam, &hover)
				}
				return
			}
			a.fontAllList(c, fp, curFam, strings.ToLower(strings.TrimSpace(s.q)), &hover)
		})
		// 手动输入:边输边预览,点「应用」才写入偏好
		if *buf != s.seen {
			s.seen = *buf
			s.prev = strings.TrimSpace(*buf)
		}
		ui.Text(c, "手动输入(支持逗号分隔回退链)").FontSize(theme.FsCap).TextColor(t.Tx3)
		ui.Row(c).Height(theme.Px(28)).PaddingX(8).Radius(t.RSm()).Background(t.Raised).Border(1, t.Bd2).Grow(1).Children(func() {
			ui.TextInputBase(c.Key(fp.key+"-manual"), buf).Grow(1).Font(theme.CodeFont()).FontSize(theme.FsCap + .5).
				Placeholder(fp.placehold).PlaceholderColor(t.Tx4)
		})
		ui.Row(c).Gap(8).Children(func() {
			kit.Btn(c, "应用字体", kit.BtnOpt{Small: true, Icon: "check"}).OnClick(func() {
				a.prefs.Set(fp.key, strings.TrimSpace(*buf))
				a.applyBehavior()
				a.flash(fp.name + "已更新")
			})
			if a.prefs.Has(fp.key) {
				kit.Btn(c, fp.resetText, kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
					a.prefs.Reset(fp.key)
					*buf, s.prev = "", ""
					a.applyBehavior()
				})
			}
		})
		prev := hover
		if prev == "" {
			prev = s.prev
		}
		if prev == "" {
			prev = cur
		}
		fp.preview(c, prev, firstFamily(prev) == curFam)
	})
}

// fontPick 点选一个字体:立刻写入偏好并生效,预览随之切到它。
func (a *App) fontPickApply(fp *fontPick, family string) {
	if family == "" {
		a.prefs.Reset(fp.key)
	} else {
		a.prefs.Set(fp.key, family)
	}
	*a.fontManualBuf(fp) = family
	fp.st.seen, fp.st.prev = family, family
	a.applyBehavior()
	a.flash(fp.name + "已更新")
}

// termFontPick 给终端选一个字体(Nerd Font 安装提示里「选用」用)。
func (a *App) termFontPick(family string) { a.fontPickApply(a.termPick(), family) }

// fontRow 画一行字体:左侧整块可点(名字用它自己的字体渲染,下面一行是安装状态),右侧是安装操作。
// 已安装的字体点一下就应用;没装的只切换预览(看回退效果)。hover 收集本帧悬停的字体,给预览用。
func (a *App) fontRow(c *ui.Context, fp *fontPick, fam, curFam string, hover *string) {
	t := a.tk
	label := fam
	if label == "" {
		label = fp.emptyName
	}
	status, known, ok := fontInstalledLabel(fam)
	on := familyNamesEqual(fam, curFam) || fam == "" && curFam == ""
	st := a.set.fontInst[fam]
	row := ui.Row(c).Key(fp.key + "-r-" + fam).Gap(4).AlignItems(ui.Center).Shrink(0)
	row.Children(func() {
		btn := ui.ButtonBase(c).Key(fp.key + "-b-" + fam).Grow(1).MinWidth(0).Height(theme.Px(40)).PaddingX(8).Gap(8).Radius(t.RSm()).
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
			fp.st.prev = fam
			if ok || fam == "" {
				a.fontPickApply(fp, fam)
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
				if fam == "" {
					sub = "未单独选择"
				}
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

// fontAllList 是「全部字体」:系统里标记为等宽的字体族,支持搜索。
func (a *App) fontAllList(c *ui.Context, fp *fontPick, curFam, q string, hover *string) {
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
		a.fontRow(c, fp, f.Name, curFam, hover)
	}
	if n == 0 {
		ui.Text(c, "没有匹配的等宽字体").FontSize(theme.FsCap).TextColor(t.Tx3).Padding(8, 4)
	}
}
