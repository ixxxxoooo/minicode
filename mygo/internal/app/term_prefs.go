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

// termFontFamily 返回终端字体;开启 nerd 回退或用户未指定 Nerd 字体时避免用不存在的 Nerd 字体名。
func (a *App) termFontFamily() string {
	fam := a.prefs.Str("term.font", "")
	if fam == "" {
		return "monospace"
	}
	if a.prefs.Bool("term.nerdFallback", true) && looksNerdFont(fam) {
		return "monospace"
	}
	return fam
}

func looksNerdFont(name string) bool {
	n := strings.ToLower(name)
	return strings.Contains(n, "nerd") || strings.Contains(n, "powerline") ||
		strings.Contains(n, "meslo") || strings.Contains(n, "fira code nf")
}
