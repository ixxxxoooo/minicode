package app

import (
	"strings"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// 认识的语言有徽标,没扩展名 / 不认识的退回线条图标。
func TestFileKindOf(t *testing.T) {
	cases := map[string]string{"app.js": "JS", "A.TSX": "TSX", "main.go": "GO", "README.md": "MD", "x.py": "PY"}
	for name, want := range cases {
		k, ok := fileKindOf(name)
		if !ok || k.label != want {
			t.Errorf("%s 应为 %s,得 %q(ok=%v)", name, want, k.label, ok)
		}
	}
	for _, name := range []string{"Makefile", ".gitignore", "a.unknownext", "noext."} {
		if _, ok := fileKindOf(name); ok {
			t.Errorf("%s 不该有徽标", name)
		}
	}
}

// 在输入框里选的权限模式要记住,新会话沿用最后一次选的(六种模式都记,含「跳过权限确认」);
// 设置里的「默认权限模式」下拉只有三档,绕过 / 自动 / 不再询问不写它;禁止绕过时 DefaultPermMode 回退,不会带出绕过。
func TestRememberPermMode(t *testing.T) {
	a, _ := newTestApp(t)
	a.rememberPermMode(core.ModePlan)
	if a.DefaultPermMode() != core.ModePlan {
		t.Fatalf("应记住计划模式,得 %v", a.DefaultPermMode())
	}
	a.rememberPermMode(core.ModeBypassPermissions)
	if a.DefaultPermMode() != core.ModeBypassPermissions {
		t.Fatalf("跳过权限确认也应被记住,得 %v", a.DefaultPermMode())
	}
	if got := permModeFromIndex(a.prefs.Int(pPermMode, 0)); got != core.ModePlan {
		t.Fatalf("设置里的默认模式不该被绕过改掉,得 %v", got)
	}
	a.prefs.Set(pPermNoByp, true)
	if a.DefaultPermMode() == core.ModeBypassPermissions {
		t.Fatal("禁止绕过权限时不该带出绕过")
	}
	a.prefs.Set(pPermNoByp, false)
	a.rememberPermMode(core.ModeAcceptEdits)
	if a.DefaultPermMode() != core.ModeAcceptEdits {
		t.Fatalf("应记住自动接受编辑,得 %v", a.DefaultPermMode())
	}
}

// 服务商分组的展开 / 折叠写进偏好,重新读入时恢复。
func TestCfgFoldedPersist(t *testing.T) {
	a, _ := newTestApp(t)
	p := a.pvLoad()
	p.folded["official"], p.folded["deepseek"] = true, true
	a.saveCfgFolded()
	if got := a.prefs.Str(prefCfgFolded, ""); got != "deepseek,official" {
		t.Fatalf("偏好内容不对:%q", got)
	}
	q := &provState{folded: map[string]bool{}}
	a.loadCfgFolded(q)
	if !q.folded["official"] || !q.folded["deepseek"] || len(q.folded) != 2 {
		t.Fatalf("恢复结果不对:%v", q.folded)
	}
	p.folded["official"] = false
	a.saveCfgFolded()
	if got := a.prefs.Str(prefCfgFolded, ""); strings.Contains(got, "official") {
		t.Fatalf("展开后不该再记折叠:%q", got)
	}
}
