package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// @author ixxxxoooo

const (
	pEdFamily = "editor.fontFamily" // 代码字体链(逗号分隔);没写 = 默认链(见 theme.DefaultCodeFamilies)
	pEdLig    = "editor.ligatures"  // 代码字体连字,默认开
)

// codeFamilyPref 返回设置里的代码字体链(已去引号、逗号规整);没设过返回空串,表示用默认链。
func (a *App) codeFamilyPref() string {
	return strings.Join(parseFontFamilies(a.prefs.Str(pEdFamily, "")), ", ")
}

// applyCodeFont 把「代码字体链 + 连字」偏好推给 theme,所有代码文字(编辑器 / 差异 / 会话代码块 / 面板)随之生效。
func (a *App) applyCodeFont() {
	theme.SetCodeFont(a.codeFamilyPref(), a.prefs.Bool(pEdLig, true))
}

// codeUIFont 返回按当前代码字体链与连字设置组好的 ui.Font。
// 参数:size 字号(px)。返回:可直接赋给 editor.State.Font 等处的字体。
func codeUIFont(size float32) ui.Font {
	return ui.Font{Family: theme.CodeFont(), Size: size, Features: theme.CodeFeatures()}
}

// codeFamilyForTerm 返回终端「没单独选过字体」时跟随的字体链(= 编辑器代码字体链,不含末尾系统等宽兜底)。
func (a *App) codeFamilyForTerm() string {
	if p := a.codeFamilyPref(); p != "" {
		return p
	}
	return theme.DefaultCodeFamilies
}
