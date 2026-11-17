package app

import (
	"path"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

var langPickLog = diag.L("ide-lang")

type langPick struct {
	label  string
	suffix string // 传给 LanguageFor 的假文件名
}

var langPicks = []langPick{
	{"跟随文件名", ""},
	{"纯文本", "file.txt"},
	{"Rust", "file.rs"},
	{"Go", "file.go"},
	{"TypeScript", "file.ts"},
	{"JavaScript", "file.js"},
	{"Python", "file.py"},
	{"Markdown", "file.md"},
	{"JSON", "file.json"},
	{"YAML", "file.yaml"},
	{"Shell", "file.sh"},
	{"HTML", "file.html"},
	{"CSS", "file.css"},
	{"SQL", "file.sql"},
}

// langMenu 状态栏语言芯片:手动指定高亮语言(空 = 按扩展名)。
func (a *App) langMenu(m *ui.Menu) {
	tb := a.ide.cur()
	if tb == nil || tb.buf == nil {
		return
	}
	for _, o := range langPicks {
		on := false
		if o.suffix == "" {
			on = tb.langPick == ""
		} else {
			on = tb.langPick == o.suffix
		}
		it := menuItem(m, o.label).Radio(on)
		if it.Chosen() {
			tb.langPick = o.suffix
			a.applyLangPick(tb)
			langPickLog.Info("手动指定语言", "path", tb.path, "语言", o.label)
			a.toast = "语言:" + o.label
		}
	}
}

func (a *App) applyLangPick(tb *ideTab) {
	if tb == nil || tb.buf == nil {
		return
	}
	suf := tb.langPick
	if suf == "" {
		suf = path.Base(tb.path)
	}
	tb.buf.SetLanguage(editor.LanguageFor(suf))
}

// langLabel 状态栏显示的语言名。
func langLabel(tb *ideTab) string {
	if tb == nil || tb.buf == nil {
		return "纯文本"
	}
	if l := tb.buf.Language(); l != nil && l.Name != "" {
		return l.Name
	}
	return "纯文本"
}
