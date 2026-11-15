package claude

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// ShortPath 路径显示:位于工作目录内的显示相对路径,否则保留末尾三段。
func ShortPath(path, cwd string) string {
	if cwd != "" && strings.HasPrefix(path, cwd) {
		if rest := strings.TrimLeft(strings.TrimPrefix(path, cwd), "/"); rest != "" {
			return rest
		}
	}
	var parts []string
	for _, p := range strings.Split(path, "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) <= 3 {
		return path
	}
	return "…/" + strings.Join(parts[len(parts)-3:], "/")
}

// BuildTool 由工具名与入参构造工具卡。
func BuildTool(id, name string, input obj, cwd string) *core.ToolCall {
	path := ""
	for _, k := range []string{"file_path", "notebook_path", "path"} {
		if p := str(input, k); p != "" {
			path = p
			break
		}
	}
	shown := ""
	if path != "" {
		shown = ShortPath(path, cwd)
	}
	var title string
	switch name {
	case "Read", "NotebookRead":
		title = "读取 " + shown
	case "Edit", "MultiEdit":
		title = "编辑 " + shown
	case "Write":
		title = "写入 " + shown
	case "NotebookEdit":
		title = "编辑笔记本 " + shown
	case "Bash":
		if d := str(input, "description"); d != "" {
			title = d
		} else {
			title = firstLine(str(input, "command"))
		}
	case "BashOutput", "TaskOutput":
		title = "读取后台输出"
	case "KillShell", "KillBash", "TaskStop":
		title = "结束后台任务"
	case "Monitor":
		d := str(input, "description")
		if d == "" {
			d = firstLine(str(input, "command"))
		}
		title = "监听 " + d
	case "Glob":
		title = "查找 " + str(input, "pattern")
	case "Grep":
		title = "搜索 " + str(input, "pattern")
	case "LS":
		title = "列出 " + shown
	case "WebFetch":
		title = "抓取 " + str(input, "url")
	case "WebSearch":
		title = "搜索网页:" + str(input, "query")
	case "ToolSearch":
		title = "查找工具:" + str(input, "query")
	case "EnterPlanMode":
		title = "进入计划模式"
	case "Skill":
		title = "技能 · " + str(input, "skill")
	case "SlashCommand":
		title = "命令 " + str(input, "command")
	case "LSP":
		title = "代码智能 · " + str(input, "operation")
	case "TaskCreate":
		title = "新建任务 " + str(input, "subject")
	case "TaskUpdate":
		if s := str(input, "status"); s != "" {
			title = "更新任务 " + str(input, "taskId") + " → " + s
		} else {
			title = "更新任务 " + str(input, "taskId")
		}
	case "TaskGet":
		title = "查看任务 " + str(input, "taskId")
	case "TaskList":
		title = "任务列表"
	case "EnterWorktree":
		title = "进入工作树 " + str(input, "name")
	case "ExitWorktree":
		title = "退出工作树"
	default:
		if strings.HasPrefix(name, "mcp__") {
			parts := strings.SplitN(name, "__", 3)
			if len(parts) == 3 {
				title = parts[1] + " · " + parts[2]
			} else {
				title = name
			}
		} else {
			title = name
		}
	}
	t := core.NewToolCall(id, name, strings.TrimSpace(title)).WithInput(input)
	switch name {
	case "BashOutput", "KillShell", "KillBash", "TaskOutput", "TaskStop":
		for _, k := range []string{"task_id", "bash_id", "shell_id"} {
			if s := str(input, k); s != "" {
				t.TaskID = s
				break
			}
		}
	}
	if path != "" {
		t.WithLocation(path)
	}
	switch name {
	case "Edit":
		if path != "" {
			old, nw := str(input, "old_string"), str(input, "new_string")
			t.WithContent(&core.DiffBlock{Path: path, Old: old, New: nw, Line: locateLine(path, cwd, old, nw)})
		}
	case "MultiEdit":
		if path != "" {
			for _, e := range arr(input, "edits") {
				em := asObj(e)
				old, nw := str(em, "old_string"), str(em, "new_string")
				t.WithContent(&core.DiffBlock{Path: path, Old: old, New: nw, Line: locateLine(path, cwd, old, nw)})
			}
		}
	case "Write":
		if path != "" {
			t.WithContent(&core.DiffBlock{Path: path, New: str(input, "content")})
		}
	}
	return t
}

func parseTodos(input obj) []core.TodoItem {
	var out []core.TodoItem
	for _, t := range arr(input, "todos") {
		tm := asObj(t)
		it := core.TodoItem{Content: str(tm, "content"), ActiveForm: str(tm, "activeForm")}
		switch str(tm, "status") {
		case "in_progress":
			it.Status = core.TodoInProgress
		case "completed":
			it.Status = core.TodoCompleted
		}
		out = append(out, it)
	}
	return out
}

func parseQuestions(input obj) []core.Question {
	var out []core.Question
	for _, q := range arr(input, "questions") {
		qm := asObj(q)
		cq := core.Question{Header: str(qm, "header"), Question: str(qm, "question"), MultiSelect: boolOf(qm, "multiSelect", false)}
		for _, o := range arr(qm, "options") {
			om := asObj(o)
			cq.Options = append(cq.Options, core.QuestionOption{Label: str(om, "label"), Description: str(om, "description")})
		}
		out = append(out, cq)
	}
	return out
}

// permissionOptions 是权限卡的选项;「总是允许」的文案取决于 claude 给出的建议。
func permissionOptions(p *Pending) []core.PermissionOption {
	label := "总是允许此类操作"
	for _, s := range p.Suggestions {
		sm := asObj(s)
		if str(sm, "type") == "setMode" && str(sm, "mode") == "acceptEdits" {
			label = "允许并自动接受编辑"
			break
		}
	}
	if label == "总是允许此类操作" {
		for _, s := range p.Suggestions {
			if str(asObj(s), "type") == "addRules" {
				label = "总是允许"
				break
			}
		}
	}
	return []core.PermissionOption{
		{ID: "allow_once", Label: "允许一次", Kind: core.AllowOnce},
		{ID: "allow_always", Label: label, Kind: core.AllowAlways},
		{ID: "reject", Label: "拒绝", Kind: core.RejectOnce},
	}
}

// locateLine 找出一处编辑在文件里的起始行号(从 1 起),找不到返回 0(界面按 1 计)。
// 工具调用事件早于执行,此时文件里还是 old,直接在原文里找 old;回放历史时文件已经改过,
// 退而在现文件里找 new。文件超过 1 MiB 或读不到都放弃,不为行号拖慢事件处理。
func locateLine(path, cwd, old, nw string) int {
	if old == "" && nw == "" {
		return 0
	}
	if !filepath.IsAbs(path) {
		if cwd == "" {
			return 0
		}
		path = filepath.Join(cwd, path)
	}
	st, err := os.Stat(path)
	if err != nil || st.IsDir() || st.Size() > 1<<20 {
		return 0
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	src := string(b)
	for _, needle := range []string{old, nw} {
		if needle == "" {
			continue
		}
		if i := strings.Index(src, needle); i >= 0 {
			return strings.Count(src[:i], "\n") + 1
		}
	}
	return 0
}
