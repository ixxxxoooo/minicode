package demo

import (
	"fmt"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func init() {
	Register(Scenario{
		ID: "tools-bash-fg-out", Title: "工具卡 · 前台命令 EvTaskOutput",
		Live: true, Build: buildScnBashFgOutput,
	})
}

// buildScnBashFgOutput 演示前台 Bash 与 EvTaskOutput 增量写入任务输出缓存。
func buildScnBashFgOutput() []Step {
	s := NewScript("tools-bash-fg-out", "工具卡 · 前台命令 EvTaskOutput")
	u := s.User("跑一条会边输出边刷新的命令。")
	taskID := "fg-npm"
	live := core.NewToolCall("fg1", "Bash", "npm run build").WithInput(map[string]any{"command": "npm run build"})
	s.Add(ms(200), core.EvAppend{Item: live})
	s.Add(ms(400), core.EvToolUpdate{ID: "fg1", Patch: core.ToolPatch{TaskID: taskID, LiveOutput: core.Str("> build\n")}})
	chunks := []string{
		"> build\n> tsc\n",
		"> build\n> tsc\nchunk 1/3\n",
		"> build\n> tsc\nchunk 2/3\n",
		"> build\n> tsc\nchunk 3/3\nDone in 12s\n",
	}
	for i, ch := range chunks {
		d := ms(700 + i*500)
		s.Add(d, core.EvToolUpdate{ID: "fg1", Patch: core.ToolPatch{LiveOutput: core.Str(ch)}})
		s.Add(d, core.EvTaskOutput{TaskID: taskID, Text: fmt.Sprintf("[stdout] line %d\n", i+1)})
	}
	s.Say("前台命令输出应随 EvTaskOutput 与 LiveOutput 同步刷新。")
	_ = u
	return s.Steps()
}
