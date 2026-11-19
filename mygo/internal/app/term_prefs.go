package app

import (
	"strconv"
	"strings"
)

// pTermCopyOnSelect 偏好:选择即复制(默认关,键与设置页 term.copyOnSelect 一致)。
const pTermCopyOnSelect = "term.copyOnSelect"

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

// termFontFamily 返回终端渲染用的字体链(MyGo 逗号分隔回退)。
func (a *App) termFontFamily() string {
	pref := a.prefs.Str("term.font", "")
	if strings.TrimSpace(pref) == "" {
		pref = a.codeFamilyForTerm() // 终端没单独选过字体:跟随编辑器的代码字体链(与 BaoCode 一致)
	}
	return buildTermFontChain(pref, a.prefs.Bool("term.nerdFallback", true))
}

func looksNerdFont(name string) bool {
	n := strings.ToLower(name)
	return strings.Contains(n, "nerd") || strings.Contains(n, "powerline") ||
		strings.Contains(n, "meslo") || strings.Contains(n, "fira code nf")
}
