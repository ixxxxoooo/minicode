package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
)

func TestSubagentCompactCardRenders(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	card := core.NewSubagent("sub1", "code-reviewer", "审查 handlers", core.SubRunning)
	card.Prompt = "只读扫描 unwrap"
	var names []string
	ui.Tester(func(c *ui.Context) {
		a.ensureTheme(c)
		a.subagentCard(c, card)
	}, func(item ui.Accessible) {
		names = append(names, item.Name)
	})
	found := false
	for _, n := range names {
		if n == "查看过程" {
			found = true
		}
	}
	if !found {
		t.Fatalf("调用卡应含「查看过程」:%v", names)
	}
}
