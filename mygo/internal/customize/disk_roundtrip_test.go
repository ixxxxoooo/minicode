package customize

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupDiskHome(t *testing.T) (home, proj string) {
	t.Helper()
	home = t.TempDir()
	proj = filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(filepath.Join(proj, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	return home, proj
}

func TestDiskRoundTripSevenKinds(t *testing.T) {
	home, proj := setupDiskHome(t)
	ds, err := OpenDisk(proj)
	if err != nil {
		t.Fatal(err)
	}

	// 命令
	cmd, err := ds.Add(Commands, "/hello", User, "打招呼")
	if err != nil {
		t.Fatal(err)
	}
	cmd.Body = "只回复一句简短问候。"
	if err := ds.Persist(Commands, cmd, cmd.Clone()); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(home, ".claude", "commands", "hello.md")
	if b, err := os.ReadFile(p); err != nil || !strings.Contains(string(b), "只回复") {
		t.Fatalf("命令文件不对:%v %q", err, b)
	}

	// 子 Agent
	ag, err := ds.Add(Agents, "greeter", User, "问候")
	if err != nil {
		t.Fatal(err)
	}
	ag.Body = "你是问候子 Agent。"
	if err := ds.Persist(Agents, ag, ag.Clone()); err != nil {
		t.Fatal(err)
	}

	// Skill
	sk, err := ds.Add(Skills, "demo-skill", User, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := ds.Persist(Skills, sk, sk.Clone()); err != nil {
		t.Fatal(err)
	}
	skillMD := filepath.Join(home, ".claude", "skills", "demo-skill", "SKILL.md")
	if _, err := os.Stat(skillMD); err != nil {
		t.Fatalf("Skill 文件未写入:%v", err)
	}

	// 规则(含未知 JSON 键保真)
	settings := filepath.Join(home, ".claude", "settings.json")
	_ = os.WriteFile(settings, []byte(`{"hooks":{},"unknownKeep":42,"permissions":{"extra":true}}`), 0o600)
	rule, err := ds.Add(Rules, "team", Project, "")
	if err != nil {
		t.Fatal(err)
	}
	rule.Body = "# 团队约定\n保持礼貌。"
	if err := ds.Persist(Rules, rule, rule.Clone()); err != nil {
		t.Fatalf("规则写盘:%v", err)
	}
	ruleMD := filepath.Join(proj, ".claude", "rules", "team.md")
	if _, err := os.Stat(ruleMD); err != nil {
		ent, _ := os.ReadDir(filepath.Join(proj, ".claude"))
		names := make([]string, 0, len(ent))
		for _, e := range ent {
			names = append(names, e.Name()+"-"+map[bool]string{true: "dir", false: "file"}[e.IsDir()])
		}
		t.Fatalf("规则文件未写入:%v .claude=%v", err, names)
	}

	// 钩子
	hk, err := ds.Add(Hooks, "Stop · *", User, "")
	if err != nil {
		t.Fatal(err)
	}
	hk.Cmd = "echo hook-ok"
	hk.Match = "*"
	hk.Event = 2
	if err := ds.Persist(Hooks, hk, hk.Clone()); err != nil {
		t.Fatal(err)
	}
	var setRoot map[string]any
	if !readJSON(settings, &setRoot) {
		t.Fatal("settings 读回失败")
	}
	if setRoot["unknownKeep"].(float64) != 42 {
		t.Fatalf("settings 未知键丢失:%v", setRoot)
	}

	// MCP 用户层 + claude.json 未知键
	claudeJSON := filepath.Join(home, ".claude.json")
	_ = os.WriteFile(claudeJSON, []byte(`{"note":"keep","mcpServers":{}}`), 0o600)
	mcp, err := ds.Add(MCP, "echo-srv", User, "echo mcp-test")
	if err != nil {
		t.Fatal(err)
	}
	mcp.Cmd = "echo mcp-test"
	if err := ds.SaveMCP(mcp, mcp.Clone(), nil); err != nil {
		t.Fatal(err)
	}
	var cj map[string]any
	if !readJSON(claudeJSON, &cj) || cj["note"] != "keep" {
		t.Fatalf("claude.json 未知键丢失:%v", cj)
	}
	mcpSrv, _ := cj["mcpServers"].(map[string]any)
	if mcpSrv["echo-srv"] == nil {
		t.Fatal("mcpServers 未写入")
	}

	// 重新打开并列出
	ds2, err := OpenDisk(proj)
	if err != nil {
		t.Fatal(err)
	}
	if ds2.Count(Commands) < 1 || ds2.Count(Agents) < 1 || ds2.Count(Skills) < 1 || ds2.Count(Rules) < 1 || ds2.Count(Hooks) < 1 || ds2.Count(MCP) < 1 {
		t.Fatalf("重载计数不对 cmd=%d ag=%d sk=%d ru=%d hk=%d mcp=%d",
			ds2.Count(Commands), ds2.Count(Agents), ds2.Count(Skills), ds2.Count(Rules), ds2.Count(Hooks), ds2.Count(MCP))
	}

	// 编辑
	cmd.Body = "问候升级为两行。"
	if err := ds2.Persist(Commands, ds2.List(Commands, All, "/hello")[0], cmd.Clone()); err != nil {
		t.Fatal(err)
	}

	// 删除命令
	it := ds2.List(Commands, All, "/hello")[0]
	if err := ds2.RemoveDisk(Commands, it); err != nil {
		t.Fatal(err)
	}
	ds2.Remove(Commands, it)
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("命令文件应已删除")
	}

	// 插件列表(只读)
	plugPath := filepath.Join(home, ".claude", "plugins", "installed_plugins.json")
	_ = os.MkdirAll(filepath.Dir(plugPath), 0o755)
	raw := `{"plugins":{"demo@market":[{"scope":"user","installPath":"` + filepath.Join(home, ".claude", "plugins", "demo") + `","version":"1.0.0"}]}}`
	_ = os.WriteFile(plugPath, []byte(raw), 0o644)
	ds3, _ := OpenDisk(proj)
	if ds3.Count(Plugins) != 1 {
		t.Fatalf("插件加载:%d", ds3.Count(Plugins))
	}
}

func TestDiskMCPProjectScope(t *testing.T) {
	_, proj := setupDiskHome(t)
	ds, err := OpenDisk(proj)
	if err != nil {
		t.Fatal(err)
	}
	it, err := ds.Add(MCP, "local-echo", Project, "echo hi")
	if err != nil {
		t.Fatal(err)
	}
	it.Cmd = "echo hi"
	it.Scope = Project
	it.Transport = "stdio"
	if err := ds.SaveMCP(it, it.Clone(), nil); err != nil {
		t.Fatal(err)
	}
	mcpPath := filepath.Join(proj, ".mcp.json")
	b, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if json.Unmarshal(b, &v) != nil {
		t.Fatal("mcp json")
	}
	srv, _ := v["mcpServers"].(map[string]any)
	if srv["local-echo"] == nil {
		t.Fatal("项目 mcpServers 缺失")
	}
}

// ~/.claude.json 里该项目的条目没有 disabledMcpServers 时不能 panic(真实用户的常见情况,曾让界面构建崩溃)。
func TestDiskMCPProjectEntryWithoutDisabledList(t *testing.T) {
	home, proj := setupDiskHome(t)
	cfg := map[string]any{"projects": map[string]any{proj: map[string]any{"allowedTools": []any{}}}}
	b, _ := json.Marshal(cfg)
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenDisk(proj); err != nil {
		t.Fatal(err)
	}
}
