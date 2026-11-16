package customize

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSplitToolsKeepsParens(t *testing.T) {
	got := SplitTools("Read, Bash(git log --oneline, -n 5)  Grep\nmcp__gh__create_issue")
	want := []string{"Read", "Bash(git log --oneline, -n 5)", "Grep", "mcp__gh__create_issue"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("拆分不对:%q", got)
	}
}

func TestValidateToolRule(t *testing.T) {
	ok := []string{"Read", "Bash(git status:*)", "mcp__github", "mcp__github__create_issue", "Read(./src/**)"}
	for _, r := range ok {
		if m := ValidateToolRule(r); m != "" {
			t.Errorf("%q 应合法:%s", r, m)
		}
	}
	bad := []string{"", "Bash(git", "Bash()", "a b", "Read,Write", "(x)", "1abc"}
	for _, r := range bad {
		if ValidateToolRule(r) == "" {
			t.Errorf("%q 应非法", r)
		}
	}
}

func TestAddAllowOrderAndDup(t *testing.T) {
	it := &Item{Allow: []string{"Bash(git diff:*)"}}
	if !it.AddAllow("Read") || it.AddAllow("Read") {
		t.Fatal("添加应成功一次、重复应拒绝")
	}
	it.AddAllow("Grep")
	if !reflect.DeepEqual(it.Allow, []string{"Read", "Grep", "Bash(git diff:*)"}) {
		t.Fatalf("内置工具排前、自定义规则在后:%q", it.Allow)
	}
}

func TestMCPToolRules(t *testing.T) {
	s := NewStore(nil, nil)
	s.items[MCP] = []*Item{{Name: "my.server", Tools: []Tool{{Name: "a"}, {Name: "b"}}}}
	got := s.MCPToolRules()
	want := []string{"mcp__my_server", "mcp__my_server__a", "mcp__my_server__b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MCP 工具规则不对:%q", got)
	}
}

func TestToolSearchValueLayers(t *testing.T) {
	home, proj := t.TempDir(), t.TempDir()
	ds := &DiskStore{home: home, proj: proj}
	write := func(p, v string) {
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		_ = os.WriteFile(p, []byte(`{"env":{"ENABLE_TOOL_SEARCH":"`+v+`"}}`), 0o644)
	}
	if on, l := ds.ToolSearchValue(); !on || l != -1 {
		t.Fatalf("没设置应按开:%v %d", on, l)
	}
	write(filepath.Join(home, ".claude", "settings.json"), "false")
	if on, l := ds.ToolSearchValue(); on || l != 0 {
		t.Fatalf("用户层关:%v %d", on, l)
	}
	write(filepath.Join(proj, ".claude", "settings.local.json"), "auto")
	if on, l := ds.ToolSearchValue(); !on || l != 2 {
		t.Fatalf("本地层应覆盖用户层:%v %d", on, l)
	}
}
