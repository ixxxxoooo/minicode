package app

import (
	"strconv"
	"strings"
)

// termScrollback 读偏好 term.scrollback(默认 10000),夹紧到合理范围。
func (a *App) termScrollback() int {
	s := strings.TrimSpace(a.prefs.Str("term.scrollback", "10000"))
	n, err := strconv.Atoi(s)
	if err != nil || n < 500 {
		n = 500
	}
	if n > 50000 {
		n = 50000
	}
	return n
}

// termFontFamily 返回终端字体。
// 「无 Nerd Font 时回退系统等宽」只在确认没装这个 Nerd 字体时才换(以前不管装没装一律换,
// 装了 Nerd Font 的用户永远用不上);字体目录没扫完时先按用户选的来。
func (a *App) termFontFamily() string {
	fam := a.prefs.Str("term.font", "")
	if fam == "" {
		return "monospace"
	}
	if a.prefs.Bool("term.nerdFallback", true) && looksNerdFont(fam) {
		if ok, known := fontInstalled(fam); known && !ok {
			return "monospace"
		}
	}
	return fam
}

func looksNerdFont(name string) bool {
	n := strings.ToLower(name)
	return strings.Contains(n, "nerd") || strings.Contains(n, "powerline") ||
		strings.Contains(n, "meslo") || strings.Contains(n, "fira code nf")
}
