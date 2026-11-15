package app

import (
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// termNerdCheck 在终端顶部提示 Nerd Font:画面里出现私有区图标(ls 图标、powerline 提示符等),
// 而当前生效的字体不是一个已装的 Nerd Font 时,图标会显示成方框。
func (a *App) termNerdCheck(c *ui.Context, tp *termPanel) {
	if tp == nil || tp.scr == nil || !a.prefs.Bool("terminal.nerdHint", true) {
		return
	}
	a.startFontScan()
	// 画面、字体、字体扫描结果都没变就不重扫整屏
	key := strconv.FormatUint(tp.scr.Ver, 10) + "|" + a.prefs.Str("term.font", "") + "|" + strconv.FormatBool(fontScan.done.Load())
	if key != tp.nerdKey {
		tp.nerdKey = key
		tp.nerdWarn = a.nerdWarning(tp)
	}
	if tp.nerdWarn == "" {
		return
	}
	t := a.tk
	bar := ui.Row(c).Key("term-nerd").Height(28).Shrink(0).PaddingX(10).Gap(8).AlignItems(ui.Center).
		Background(t.WarnSoft).BorderWidth(0, 0, 1, 0).BorderColor(t.Warn).Children(func() {
		ui.Text(c, tp.nerdWarn).FontSize(theme.FsCap).TextColor(t.Warn).Grow(1).MinWidth(0).SingleLine()
		kit.Btn(c, "终端设置", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
			a.openSettings("terminal")
		})
		kit.Btn(c, "不再提示", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
			tp.nerdWarn = ""
			a.prefs.Set("terminal.nerdHint", false)
		})
	})
	tp.addOverlay(bar.Bounds())
}

// nerdWarning 返回提示文案;空 = 不需要提示。
func (a *App) nerdWarning(tp *termPanel) string {
	if !hasPrivateGlyphs(tp) {
		return ""
	}
	fam := a.prefs.Str("term.font", "")
	if looksNerdFont(fam) {
		if ok, known := fontInstalled(fam); !known || ok {
			return "" // 选的是 Nerd Font 且已装(或还没扫完):图标能显示
		}
	}
	if nerdFontInstalled() {
		return "画面里有 Nerd Font 图标,当前字体显示成方框。已检测到你装了 Nerd Font,可在「终端设置」里选用。"
	}
	return "画面里有 Nerd Font 图标,当前字体显示成方框。安装 Nerd Font(如 brew install --cask font-meslo-lg-nerd-font)后在「终端设置」里选用。"
}

// hasPrivateGlyphs 画面里是否有私有区字符(Nerd Font 图标的码位:BMP 私有区与补充私有区 A)。
func hasPrivateGlyphs(tp *termPanel) bool {
	for _, ln := range tp.scr.Lines {
		for _, run := range ln {
			for _, r := range run.Text {
				if (r >= 0xE000 && r <= 0xF8FF) || (r >= 0xF0000 && r <= 0xFFFFD) {
					return true
				}
			}
		}
	}
	return false
}
