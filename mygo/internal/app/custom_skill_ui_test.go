package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/customize"
)

func TestCuSkillFileListColumn(t *testing.T) {
	a, tt := newTestApp(t)
	a.SeedDemoExtras()
	a.screen = ScreenCustomize
	a.cu.cat = customize.Skills
	// 直接构造一个带多文件的 Skill,避免依赖演示数据里是否包含它
	it := &customize.Item{
		Name: "ok-skill", Scope: customize.User, On: true, Path: "~/.claude/skills/ok",
		Files: []customize.File{{Name: "SKILL.md", Active: true, Bytes: 1200}, {Name: "ref.md", Bytes: 340}, {Name: "examples/demo.sh", Bytes: 89}},
		Body:  "---\nname: ok-skill\n---\n\n# OK\n\n短正文。",
	}
	a.cust = customize.NewStore(map[customize.Cat][]*customize.Item{customize.Skills: {it}}, nil)
	a.cuSelect(it)
	tt.Frame()
	a.drainUI()
	tt.Frame()
	if a.cu.skillFile != "SKILL.md" {
		t.Fatalf("应默认选中 SKILL.md,得到 %q", a.cu.skillFile)
	}
	if !tt.HasText("ref.md") || !tt.HasText("1 KB") {
		t.Fatalf("应纵向列出文件与大小:%v", tt.Texts())
	}
	if err := tt.Click("ref.md"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	a.drainUI()
	tt.Frame()
	if a.cu.skillFile != "ref.md" {
		t.Fatalf("应选中 ref.md,得到 %q", a.cu.skillFile)
	}
}
