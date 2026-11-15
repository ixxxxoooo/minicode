package customize

import "testing"

func mk() *Store {
	return NewStore(map[Cat][]*Item{
		MCP: {
			{Name: "github", Desc: "stdio · 已连接", Scope: User, On: true, Status: StOK, Tools: []Tool{{"a", true}, {"b", false}}},
			{Name: "postgres", Desc: "连接失败", Scope: Local, Status: StErr, Cmd: "npx pg", Transport: "stdio"},
		},
		Commands: {{Name: "/review", Scope: Project, On: true}},
		Agents:   {{Name: "reader", Scope: User, Allow: []string{"Read", "Grep", "Glob"}}},
	}, []*Market{{Name: "frontend-design"}, {Name: "code-review", Installed: true}})
}

func TestListFilter(t *testing.T) {
	s := mk()
	if len(s.List(MCP, All, "")) != 2 || len(s.List(MCP, Local, "")) != 1 || len(s.List(MCP, All, "GITHUB")) != 1 || len(s.List(MCP, User, "postgres")) != 0 {
		t.Fatal("范围与关键字筛选不对")
	}
	if s.Count(MCP) != 2 || s.Count(Hooks) != 0 {
		t.Fatal("计数不对")
	}
}

func TestAddValidate(t *testing.T) {
	s := mk()
	for _, tc := range []struct {
		c    Cat
		name string
		ok   bool
	}{
		{MCP, "", false}, {MCP, "has space", false}, {MCP, "github", false}, {MCP, "figma", true},
		{Commands, "review", false}, {Commands, "/review", false}, {Commands, "/test", true},
		{Hooks, "PostToolUse · Edit", true}, {Plugins, "x", false},
	} {
		_, err := s.Add(tc.c, tc.name, All, "x")
		if (err == nil) != tc.ok {
			t.Errorf("%s %q:err=%v", tc.c, tc.name, err)
		}
	}
	it, _ := s.Add(MCP, "remote", Project, "https://x.dev/mcp")
	if it.Transport != "http" || it.Scope != Project || it.Status != StErr {
		t.Fatalf("新 MCP 不对:%+v", it)
	}
	it.Reconnect()
	if it.Status != StOK {
		t.Fatal("有地址应能连上")
	}
	s.Remove(MCP, it)
	if s.Has(MCP, "remote", nil) {
		t.Fatal("应已删除")
	}
}

func TestToolsAndPreset(t *testing.T) {
	s := mk()
	g := s.List(MCP, All, "github")[0]
	g.ToggleTool(1)
	g.ToggleTool(9) // 越界不动
	if !g.Tools[1].On {
		t.Fatal("工具应被打开")
	}
	a := s.List(Agents, All, "")[0]
	if a.Preset() != 0 {
		t.Fatal("只读预设")
	}
	a.ToggleAllow("Bash")
	a.ToggleAllow("Edit")
	if a.Preset() != 1 || a.Allow[3] != "Edit" || a.Allow[4] != "Bash" {
		t.Fatalf("应按工具表顺序插入:%v", a.Allow)
	}
	a.SetPreset(2)
	if a.Preset() != 2 {
		t.Fatal("全部预设")
	}
	a.ToggleAllow("Bash")
	if len(a.Allow) != len(AllTools)-1 {
		t.Fatal("应能取消一个工具")
	}
}

func TestPlugins(t *testing.T) {
	s := mk()
	it, err := s.Install(s.Market[0])
	if err != nil || !s.Market[0].Installed || s.Count(Plugins) != 1 {
		t.Fatalf("安装失败:%v", err)
	}
	if _, err := s.Install(s.Market[1]); err == nil {
		t.Fatal("已安装的不能再装")
	}
	it.Upd = "2.0.0"
	it.Update()
	if it.Ver != "2.0.0" || it.Upd != "" {
		t.Fatal("更新不对")
	}
	s.Uninstall(it)
	if s.Count(Plugins) != 0 || s.Market[0].Installed {
		t.Fatal("卸载应同步市场状态")
	}
}

func TestDraftSave(t *testing.T) {
	s := mk()
	a := s.List(Agents, All, "")[0]
	d := a.Clone()
	d.Name = "writer"
	d.ToggleAllow("Edit")
	if a.Name != "reader" || len(a.Allow) != 3 {
		t.Fatal("改草稿不应影响原条目")
	}
	if err := s.Save(Agents, a, d); err != nil || a.Name != "writer" || len(a.Allow) != 4 {
		t.Fatalf("保存失败:%v", err)
	}
	d2 := a.Clone()
	d2.Name = "has space"
	if err := s.Save(Agents, a, d2); err == nil || a.Name != "writer" {
		t.Fatal("非法名称不应保存")
	}
	h := &Item{Name: "Stop", Timeout: 5}
	s.items[Hooks] = []*Item{h}
	d3 := h.Clone()
	d3.Timeout = 0
	if s.Save(Hooks, h, d3) == nil {
		t.Fatal("超时 0 应被拒绝")
	}
}
