package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// termNerdCheck 检测画面里私有区符号,若与常见 Nerd Font 缺字一并出现则提示安装字体。
func (a *App) termNerdCheck(c *ui.Context, tp *termPanel) {
	if tp == nil || tp.scr == nil || !a.prefs.Bool("terminal.nerdHint", true) {
		return
	}
	if tp.nerdWarn == "" {
		tp.nerdWarn = scanNerdGap(tp)
	}
	if tp.nerdWarn == "" {
		return
	}
	t := a.tk
	ui.Row(c).Key("term-nerd").Height(28).Shrink(0).PaddingX(10).Gap(8).AlignItems(ui.Center).
		Background(t.WarnSoft).BorderWidth(0, 0, 1, 0).BorderColor(t.Warn).Children(func() {
		ui.Text(c, tp.nerdWarn).FontSize(theme.FsCap).TextColor(t.Warn).Grow(1).SingleLine()
		kit.Btn(c, "知道了", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
			tp.nerdWarn = ""
			a.prefs.Set("terminal.nerdHint", false)
		})
	})
}

func scanNerdGap(tp *termPanel) string {
	box := 0
	priv := 0
	for _, ln := range tp.scr.Lines {
		for _, run := range ln {
			for _, r := range run.Text {
				if r == '\uFFFD' || r == '□' {
					box++
				}
				if r >= 0xE000 && r <= 0xF8FF {
					priv++
				}
			}
		}
	}
	if box >= 3 && priv > 0 {
		return "终端字体可能缺少 Nerd Font 字形(画面出现方框)。可在「终端设置」里换 Nerd Font 或 Meslo 等等宽字体。"
	}
	if priv > 12 && box > 0 {
		return "检测到私有区图标字符但字体未完全覆盖,建议安装 Nerd Font。"
	}
	return ""
}
