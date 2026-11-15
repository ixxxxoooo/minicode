package claude

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func TestTranslateTaskEvents(t *testing.T) {
	cases := []struct {
		name string
		line string
		want func(t *testing.T, evs []core.Event, st *core.State)
	}{
		{
			name: "task_started",
			line: `{"type":"system","subtype":"task_started","task_id":"tk1","tool_use_id":"u1","task_type":"local_bash","description":"dev server","run_id":"run-abc","pid":4242,"is_backgrounded":true}`,
			want: func(t *testing.T, evs []core.Event, st *core.State) {
				if len(evs) != 1 {
					t.Fatalf("事件数=%d", len(evs))
				}
				e, ok := evs[0].(core.EvTaskStarted)
				if !ok || e.Task.TaskID != "tk1" || e.Task.RunID != "run-abc" || e.Task.ProcessID != 4242 {
					t.Fatalf("task_started 解析不对:%#v", e)
				}
				st.Apply(e)
				if len(st.Tasks) != 1 || st.Tasks[0].Kind != core.TaskShell {
					t.Fatalf("归约后任务:%#v", st.Tasks)
				}
			},
		},
		{
			name: "task_progress",
			line: `{"type":"system","subtype":"task_progress","task_id":"tk1","last_tool_name":"Bash","summary":"编译中","usage":{"total_tokens":900,"tool_uses":2}}`,
			want: func(t *testing.T, evs []core.Event, st *core.State) {
				st.Apply(core.EvTaskStarted{Task: core.NewBackgroundTask(0, "tk1", core.TaskShell, "x")})
				for _, e := range evs {
					st.Apply(e)
				}
				tk := st.Tasks[0]
				if tk.LastTool != "Bash" || tk.Summary != "编译中" || tk.Tokens != 900 || tk.ToolUses != 2 {
					t.Fatalf("progress:%#v", tk)
				}
			},
		},
		{
			name: "task_notification",
			line: `{"type":"system","subtype":"task_notification","task_id":"tk1","status":"completed","summary":"完成","output_file":"/tmp/out.log"}`,
			want: func(t *testing.T, evs []core.Event, st *core.State) {
				st.Apply(core.EvTaskStarted{Task: core.NewBackgroundTask(0, "tk1", core.TaskAgent, "调研")})
				for _, e := range evs {
					st.Apply(e)
				}
				if st.Tasks[0].Status != core.TaskCompleted || st.Tasks[0].Summary != "完成" {
					t.Fatalf("notification:%#v", st.Tasks[0])
				}
			},
		},
		{
			name: "background_tasks_changed",
			line: `{"type":"system","subtype":"background_tasks_changed","tasks":[{"task_id":"bg1","task_type":"local_agent","description":"审查","is_backgrounded":true},{"task_id":"sh2","task_type":"local_bash","description":"watch","is_backgrounded":true}]}`,
			want: func(t *testing.T, evs []core.Event, st *core.State) {
				for _, e := range evs {
					st.Apply(e)
				}
				if len(st.Tasks) != 2 {
					t.Fatalf("快照任务数=%d", len(st.Tasks))
				}
			},
		},
	}
	tr := NewTranslator("/w")
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := parseLine(c.line)
			if v == nil {
				t.Fatal("JSON 无效")
			}
			evs := tr.Handle(v)
			c.want(t, evs, core.NewState())
		})
	}
}
