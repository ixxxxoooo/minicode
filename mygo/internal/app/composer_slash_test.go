package app

import "testing"

// 只输入一个 / 就要打开补全并列出全部命令;输入空格(开始写参数)后关闭。
func TestSlashOpensOnBareSlash(t *testing.T) {
	a, _ := sessionApp(t, "flow-gaps-b4")
	for _, c := range []struct {
		draft string
		open  bool
	}{
		{"", false},
		{"/", true},
		{"/co", true},
		{"  /", true},
		{"/compact ", false},
		{"hello /", false},
	} {
		a.draft = c.draft
		if got := a.slashOpen(); got != c.open {
			t.Fatalf("draft=%q slashOpen=%v,期望 %v", c.draft, got, c.open)
		}
	}
	a.draft = "/"
	if len(a.slashRows()) == 0 {
		t.Fatal("单个 / 应列出全部可用命令")
	}
}
