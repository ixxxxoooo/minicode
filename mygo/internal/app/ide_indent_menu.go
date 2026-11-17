package app

import (
	"github.com/egoist/mygo/ui"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

var indentLog = diag.L("ide-indent")

// indentMenu 是状态栏「空格:4 / Tab」点开的菜单(设计稿 ide-nav 三区清单):
// 跟随文件(自动检测)/ 空格 2 / 空格 4 / 制表符;选择写偏好 editor.indent(0 = 空格 2,1 = 空格 4,2 = Tab,没设置 = 自动)。
func (a *App) indentMenu(m *ui.Menu) {
	has := a.prefs.Has(pEdIndent)
	cur := a.prefs.Int(pEdIndent, 1)
	opts := []struct {
		label string
		val   int // -1 = 自动
		on    bool
	}{
		{"跟随文件(自动检测)", -1, !has},
		{"空格:2", 0, has && cur == 0},
		{"空格:4", 1, has && cur == 1},
		{"空格:8", 3, has && cur == 3},
		{"制表符(Tab)", 2, has && cur == 2},
	}
	for _, o := range opts {
		if menuItem(m, o.label).Radio(o.on).Chosen() {
			if o.val < 0 {
				a.prefs.Reset(pEdIndent)
			} else {
				a.prefs.Set(pEdIndent, o.val)
			}
			indentLog.Info("切换缩进方式", "选择", o.label)
			a.toast = "缩进:" + o.label
		}
	}
}
