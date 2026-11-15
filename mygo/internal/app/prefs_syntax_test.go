package app

import (
	"testing"

	"github.com/egoist/mygo/ui/editor"
)

// 设置「语法高亮」关闭后,编辑器主题里的词法着色全部取正文色;开启时保持主题原样。
func TestSyntaxPrefStripsTokens(t *testing.T) {
	a, _ := newTestApp(t)
	mk := func() *ideTab {
		st := &editor.State{Buffer: editor.NewBuffer("package x")}
		st.Theme = a.editorTheme() // 与每帧开头一致:先取共享主题
		return &ideTab{st: st}
	}
	plain := func(th *editor.Theme) bool { // 每一种词法着色都等于正文色(零值会被库补成默认色,不算)
		for _, tk := range th.Tokens {
			if tk.Color != th.Text || tk.Bold || tk.Italic {
				return false
			}
		}
		return true
	}

	on := mk()
	a.prefs.Set(pEdSyntax, true)
	a.applyEditorPrefs(on)
	if plain(on.st.Theme) {
		t.Fatal("开启时不应抹掉词法着色")
	}

	off := mk()
	a.prefs.Set(pEdSyntax, false)
	a.applyEditorPrefs(off)
	if !plain(off.st.Theme) {
		t.Fatal("关闭后词法着色应全部取正文色")
	}
	if plain(a.editorTheme()) {
		t.Fatal("不能改动共享的基础主题")
	}
}
