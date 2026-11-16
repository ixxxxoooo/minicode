package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/customize"
)

func TestCuOpenPathHomeSkill(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	skillDir := filepath.Join(home, ".claude", "skills", "demo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	skillMD := filepath.Join(skillDir, "SKILL.md")
	_ = os.WriteFile(skillMD, []byte("# demo\n"), 0o644)

	a, _ := newTestApp(t)
	a.demoExtra = false
	a.proj = t.TempDir()
	ds, err := customize.OpenDisk(a.proj)
	if err != nil {
		t.Fatal(err)
	}
	a.cu.disk, a.cust = ds, ds.Store
	a.cu.cat = customize.Skills
	a.cuSelect(&customize.Item{Name: "demo", Path: "~/.claude/skills/demo", Scope: customize.User})
	a.cuOpenPath(a.cu.sel.Path)
	if a.mode != ModeIDE {
		t.Fatalf("应进入 IDE:%v", a.mode)
	}
	if cur := a.ide.cur(); cur == nil || cur.path != skillMD {
		t.Fatalf("应打开 SKILL.md:%v", cur)
	}
}

func TestCuRunHookTimeoutAndExit(t *testing.T) {
	a, _ := newTestApp(t)
	a.cu.hookOut = ""
	a.cuRunHook("sleep 5", 1)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && !strings.Contains(a.cu.hookOut, "超时") {
		time.Sleep(50 * time.Millisecond)
	}
	if !strings.Contains(a.cu.hookOut, "超时") {
		t.Fatalf("应超时:%q", a.cu.hookOut)
	}
	a.cu.hookOut = ""
	a.cuRunHook("exit 7", 10)
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && !strings.HasPrefix(a.cu.hookOut, "退出码") {
		time.Sleep(50 * time.Millisecond)
	}
	if !strings.HasPrefix(a.cu.hookOut, "退出码 7") {
		t.Fatalf("应记录退出码:%q", a.cu.hookOut)
	}
}

func TestCuAuthPrefillsTerminal(t *testing.T) {
	a, _ := newTestApp(t)
	a.demoExtra = false
	a.proj = t.TempDir()
	it := &customize.Item{Name: "gh", Status: customize.StWarn}
	a.cuAuthReal(it)
	if a.ide.tpc().initCmd == "" || !strings.Contains(a.ide.tpc().initCmd, "mcp login") {
		t.Fatalf("终端命令未预填:%q", a.ide.tpc().initCmd)
	}
	if !strings.Contains(a.ide.tpc().initCmd, "gh") {
		t.Fatal("应包含服务器名")
	}
}

func TestPluginInstallPhaseFailure(t *testing.T) {
	a, _ := newTestApp(t)
	a.demoExtra = false
	st := customize.NewStore(nil, []*customize.Market{{Name: "demo-plug"}})
	m := st.Market[0]
	a.cu.plug = &pluginInstall{name: m.Name, phase: 3, err: "市场不可达"}
	if a.cu.plug.phase != 3 {
		t.Fatal("应保持失败态")
	}
}

func TestCuDraftTryCommandUsesBody(t *testing.T) {
	st := customize.NewStore(map[customize.Cat][]*customize.Item{
		customize.Commands: {{Name: "/x", Body: "saved", Scope: customize.User}},
	}, nil)
	a, _ := newTestApp(t)
	it := st.List(customize.Commands, customize.All, "")[0]
	a.cuSelect(it)
	a.cu.draft.Body = "draft-only"
	a.cu.dOf = it
	if body := strings.TrimSpace(a.cuDraftItem(it).Body); body != "draft-only" {
		t.Fatalf("试运行应读草稿:%q", body)
	}
}