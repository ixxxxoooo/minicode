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
	st := a.cuStore()
	for _, it := range st.List(customize.Skills, customize.All) {
		if it.Name == "ok-skill" {
			a.cuSelect(it)
			break
		}
	}
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
