package demo

import (
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func pressureBuilds() map[string]func() []Step {
	return map[string]func() []Step{
		"long":               buildLongSession,
		"press-huge-reply":   buildPressHugeReply,
		"press-cmd-output":   buildPressCmdOutput,
		"press-huge-diff":    buildPressHugeDiff,
		"press-mass-tools":   buildPressMassTools,
		"press-deep-sub":     buildPressDeepSub,
		"press-stream-flood": buildPressStreamFlood,
		"press-huge-user":    buildPressHugeUser,
	}
}

func TestPressureBuildIDsUnique(t *testing.T) {
	for name, fn := range pressureBuilds() {
		start := time.Now()
		steps := fn()
		if d := time.Since(start); d > 8*time.Second {
			t.Errorf("%s: 构建过慢 %v", name, d)
		}
		if len(steps) < minSteps(name) {
			t.Errorf("%s: 步数过少 %d", name, len(steps))
		}
		ids := collectItemIDs(steps)
		seen := map[string]int{}
		for _, id := range ids {
			if id == "" {
				continue
			}
			seen[id]++
			if seen[id] > 1 {
				t.Fatalf("%s: 重复条目 id %q", name, id)
			}
		}
	}
}

func minSteps(name string) int {
	switch name {
	case "long":
		return 16400
	case "press-mass-tools":
		return 1600
	case "press-stream-flood":
		return 12000
	default:
		return 8
	}
}

func collectItemIDs(steps []Step) []string {
	var ids []string
	for _, st := range steps {
		switch e := st.Ev.(type) {
		case core.EvAppend:
			if e.Item != nil {
				ids = append(ids, e.Item.ItemID())
			}
		case core.EvTodos:
			ids = append(ids, e.ID)
		}
	}
	return ids
}
