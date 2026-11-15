package demo

import (
	"fmt"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// 构造说明:
// - buildLongSession 生成 2000 个完整回合(用户 + 步骤 + 助手 + 统计),瞬时回放。
// - 条目 id 前缀 L{回合}-{后缀};预估顶层时间线条目约 2.8–3.5 万(含工具与子 Agent 嵌套项在归约后更多)。
// - 每步用 stepBuf 直接追加已完成条目,避免流式分块;预估剧本步数约 3.2–3.8 万。

const longTurns = 2000

type longBuild struct {
	buf *stepBuf
	rng *detRand
}

func buildLongSession() []Step {
	const estSteps = 36000
	b := &longBuild{buf: &stepBuf{}, rng: newDetRand(0x4C4F4E47)}
	b.buf.cap(estSteps)
	b.buf.ev(core.EvReady{SessionID: "demo-long", Models: core.DefaultModels(), Model: "sonnet", Mode: core.ModeDefault})
	b.buf.ev(core.EvTitleChanged{Title: "2000 轮长会话:滚动与加载"})
	for round := 1; round <= longTurns; round++ {
		switch {
		case round%200 == 198:
			b.turnFail(round)
		case round%200 == 199:
			b.turnInterrupt(round)
		case round%50 == 0:
			b.turnMilestone(round)
		default:
			b.turnNormal(round)
		}
	}
	return b.buf.out()
}

func (b *longBuild) uid(round int) string    { return fmt.Sprintf("L%d-u", round) }
func (b *longBuild) tid(round int) string    { return fmt.Sprintf("L%d-t", round) }
func (b *longBuild) aid(round int) string    { return fmt.Sprintf("L%d-a", round) }
func (b *longBuild) kid(round, k int) string { return fmt.Sprintf("L%d-k%d", round, k) }

func (b *longBuild) turnNormal(round int) {
	uid := b.uid(round)
	path := fmt.Sprintf("src/pkg%d/mod%d.go", round%40, round%17)
	msg := b.userText(round, false)
	b.buf.ev(core.EvAppend{Item: &core.User{ID: uid, Text: msg, Files: []string{path}}})
	b.buf.ev(core.EvTurnStarted{})
	k := 0
	next := func() string { k++; return b.kid(round, k) }

	switch round % 20 {
	case 0:
		b.buf.think(b.tid(round), fmt.Sprintf("第 %d 轮：先读 `%s` 理清结构。", round, path), uint64(2+round%5))
		tc := core.NewToolCall(next(), "Read", "读取 "+path).WithInput(map[string]any{"file_path": path}).WithLocation(path)
		b.buf.toolDone(tc, 120+int64(round%90), &core.TextBlock{Text: genLines(12+round%30, "// ")})
	case 1:
		tc := core.NewToolCall(next(), "Grep", "搜索 handler").WithInput(map[string]any{"pattern": "handler", "path": "src/"})
		out := fmt.Sprintf("Found %d files\n%s\nsrc/server.go\n", 3+round%5, path)
		b.buf.toolDone(tc, 200, &core.TextBlock{Text: out})
	case 2:
		old, neu := genDiffBody(8+round%6, path)
		tc := core.NewToolCall(next(), "Edit", "编辑 "+path).WithInput(map[string]any{"file_path": path}).WithLocation(path)
		b.buf.toolDone(tc, 400, &core.DiffBlock{Path: path, Old: old, New: neu, Line: 10 + round%20})
	case 3:
		body := genLines(6, "export fn ")
		tc := core.NewToolCall(next(), "Write", "写入 "+path).WithInput(map[string]any{"file_path": path}).WithLocation(path)
		b.buf.toolDone(tc, 300, &core.TextBlock{Text: body})
	case 4:
		tc := core.NewToolCall(next(), "Bash", "运行测试").WithInput(map[string]any{"command": "cargo test -q"})
		b.buf.toolDone(tc, 1800, &core.TerminalBlock{Command: "cargo test -q", Output: "test ok\n", ExitCode: core.Int(0)})
	case 5:
		tc := core.NewToolCall(next(), "Bash", "运行 lint").WithInput(map[string]any{"command": "npm run lint"})
		b.buf.toolFail(tc, 900, fmt.Sprintf("exit 1 · round %d", round))
	case 6:
		tc := core.NewToolCall(next(), "mcp__github__get_pr", "github · get_pr").WithInput(map[string]any{"number": round})
		b.buf.toolDone(tc, 500, &core.TextBlock{Text: fmt.Sprintf(`{"title":"PR #%d","state":"open"}`, round)})
	case 7:
		tc := core.NewToolCall(next(), "WebFetch", "抓取文档").WithInput(map[string]any{"url": "https://example.com/docs"})
		b.buf.toolDone(tc, 700, &core.TextBlock{Text: "# API\n\n中英 mixed doc chunk."})
	case 8:
		tc := core.NewToolCall(next(), "WebSearch", "联网搜索").WithInput(map[string]any{"query": "rust router normalize"})
		b.buf.toolDone(tc, 900, &core.TextBlock{Text: "1. Blog post …\n2. Issue thread …"})
	case 9:
		b.buf.ev(core.EvTodos{ID: fmt.Sprintf("L%d-td", round), Items: []core.TodoItem{
			{Content: "梳理入口", Status: core.TodoCompleted},
			{Content: fmt.Sprintf("改 %s", path), Status: core.TodoInProgress, ActiveForm: "编辑"},
			{Content: "补测试", Status: core.TodoPending},
		}})
	case 10:
		sid := fmt.Sprintf("L%d-sub", round)
		b.buf.ev(core.EvAppend{Item: core.NewSubagent(sid, "explore", "子 Agent 探索", core.SubRunning)})
		rid := next()
		tc := core.NewToolCall(rid, "Read", "读取 "+path).WithInput(map[string]any{"file_path": path}).WithLocation(path)
		b.buf.ev(core.EvAppend{Parent: sid, Item: tc})
		b.buf.ev(core.EvToolUpdate{Parent: sid, ID: rid, Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), AppendContent: []core.ToolContent{&core.TextBlock{Text: "…"}}}})
		b.buf.ev(core.EvSubagentFinished{ID: sid, Status: core.SubCompleted, Summary: "完成探索", ToolCount: 1, Tokens: 800})
	case 11:
		bash := core.NewToolCall(next(), "Bash", "写配置").WithInput(map[string]any{"command": "echo patch"})
		b.buf.ev(core.EvAppend{Item: &core.PermissionRequest{ID: bash.ID, Tool: bash, Resolved: "allow", Options: []core.PermissionOption{
			{ID: "allow", Label: "允许一次", Kind: core.AllowOnce},
		}}})
		b.buf.ev(core.EvToolUpdate{ID: bash.ID, Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), AppendContent: []core.ToolContent{&core.TerminalBlock{Command: "echo patch", Output: "ok\n", ExitCode: core.Int(0)}}}})
	case 12:
		b.buf.ev(core.EvAppend{Item: &core.QuestionRequest{
			ID: fmt.Sprintf("L%d-q", round), Questions: Questions(), Answered: true,
			Answers: [][]string{{"忽略末尾斜杠"}, {"加一个开关"}},
		}})
	case 13:
		ok := true
		b.buf.ev(core.EvAppend{Item: &core.PlanApproval{ID: fmt.Sprintf("L%d-p", round), Markdown: PlanMarkdown, Approved: &ok}})
	case 14:
		b.buf.ev(core.EvNotice{Level: core.NoticeInfo, Text: fmt.Sprintf("信息 · 第 %d 轮上下文已压缩", round)})
	case 15:
		b.buf.ev(core.EvNotice{Level: core.NoticeWarning, Text: fmt.Sprintf("警告 · 第 %d 轮缓存命中率偏低", round)})
	case 16:
		b.buf.ev(core.EvNotice{Level: core.NoticeError, Text: fmt.Sprintf("错误(已恢复) · 第 %d 轮瞬时限流", round)})
	case 17:
		files := []core.ChangedFile{{Path: path, Added: 2 + round%5, Removed: 1}}
		b.buf.ev(core.EvAppend{Item: &core.Changes{ID: fmt.Sprintf("L%d-ch", round), Files: files}})
	case 18:
		tid := fmt.Sprintf("L%d-bg", round)
		task := core.NewBackgroundTask(int64(round)*1000, tid, core.TaskShell, "后台 watch")
		task.Backgrounded = true
		b.buf.ev(core.EvTaskStarted{Task: task})
		b.buf.ev(core.EvTaskNotification{TaskID: tid, Status: core.TaskCompleted, Summary: "watch 正常退出"})
	case 19:
		tc := core.NewToolCall(next(), "Read", "查看截图").WithInput(map[string]any{"file_path": "screenshots/ui.png"})
		b.buf.toolDone(tc, 90, &core.ImageBlock{Label: "界面截图", Path: "screenshots/ui.png"})
	default:
		b.buf.think(b.tid(round), "…", 1)
	}
	if round%120 == 0 {
		b.buf.ev(core.EvMcpServers{Servers: demoMcpServers()})
	}
	b.buf.say(b.aid(round), b.replyText(round, false))
	b.buf.endTurn(uid, round, core.StopEndTurn)
}

func (b *longBuild) turnMilestone(round int) {
	uid := b.uid(round)
	b.buf.ev(core.EvAppend{Item: &core.User{ID: uid, Text: b.userText(round, true)}})
	b.buf.ev(core.EvTurnStarted{})
	b.buf.think(b.tid(round), genMarkdownBulk(400+round%200, "里程碑思考"), uint64(8+round%10))
	k := 0
	names := []string{"Read", "Grep", "Edit", "Bash", "WebSearch", "Write", "mcp__github__get_pr", "WebFetch"}
	for i := 0; i < 10; i++ {
		k++
		id := b.kid(round, k)
		path := fmt.Sprintf("src/milestone/%d_%d.rs", round, i)
		name := names[i%len(names)]
		tc := core.NewToolCall(id, name, fmt.Sprintf("%s · %s", name, path)).WithInput(map[string]any{"file_path": path})
		var content core.ToolContent = &core.TextBlock{Text: genLines(20+i*3, "// ")}
		if name == "Edit" {
			o, n := genDiffBody(12, path)
			content = &core.DiffBlock{Path: path, Old: o, New: n, Line: 1}
		}
		if name == "Bash" {
			content = &core.TerminalBlock{Command: "cargo test", Output: genLines(30, "ok "), ExitCode: core.Int(0)}
		}
		b.buf.toolDone(tc, 200+int64(i*50), content)
	}
	chg := make([]core.ChangedFile, 0, 8)
	for i := 0; i < 8; i++ {
		chg = append(chg, core.ChangedFile{Path: fmt.Sprintf("src/milestone/f%d.go", i), Added: 10 + i, Removed: i})
	}
	b.buf.ev(core.EvAppend{Item: &core.Changes{ID: fmt.Sprintf("L%d-chM", round), Files: chg}})
	b.buf.say(b.aid(round), genMarkdownBulk(1200+round%500, fmt.Sprintf("里程碑 %d", round)))
	b.buf.endTurn(uid, round, core.StopEndTurn)
}

func (b *longBuild) turnFail(round int) {
	uid := b.uid(round)
	b.buf.ev(core.EvAppend{Item: &core.User{ID: uid, Text: fmt.Sprintf("第 %d 轮(失败演练)：跑 CI", round)}})
	b.buf.ev(core.EvTurnStarted{})
	tc := core.NewToolCall(b.kid(round, 1), "Bash", "CI 管线").WithInput(map[string]any{"command": "npm run ci"})
	b.buf.toolFail(tc, 4000, genLines(25, "ERR "))
	b.buf.ev(core.EvNotice{Level: core.NoticeError, Text: "请求失败：远端 Runner 拒绝执行(演示)"})
	b.buf.say(b.aid(round), "CI 失败，已记录日志片段。")
	b.buf.ev(core.EvTurnStats{ID: uid, Stats: *DefaultStats(round)})
	b.buf.ev(core.EvDuration{ID: uid, Secs: uint64(30 + round%20)})
	b.buf.ev(core.EvTurnEnded{Reason: core.StopError})
}

func (b *longBuild) turnInterrupt(round int) {
	uid := b.uid(round)
	b.buf.ev(core.EvAppend{Item: &core.User{ID: uid, Text: fmt.Sprintf("第 %d 轮(中断演练)：长任务", round)}})
	b.buf.ev(core.EvTurnStarted{})
	tc := core.NewToolCall(b.kid(round, 1), "Bash", "sleep 120").WithInput(map[string]any{"command": "sleep 120"})
	b.buf.ev(core.EvAppend{Item: tc})
	b.buf.say(b.aid(round), "——")
	b.buf.ev(core.EvDuration{ID: uid, Secs: 12})
	b.buf.ev(core.EvTurnEnded{Reason: core.StopCancelled})
}

func (b *longBuild) userText(round int, big bool) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("回合 %d · ", round))
	sb.WriteString(b.rng.pick("请继续", "检查改动", "补测试", "optimize hot path", "对齐设计稿"))
	if big {
		sb.WriteString("\n\n")
		sb.WriteString(genMarkdownBulk(200, "用户补充"))
	}
	return sb.String()
}

func (b *longBuild) replyText(round int, big bool) string {
	if big {
		return genMarkdownBulk(600, "回复")
	}
	n := 40 + round%80 + b.rng.intn(60)
	return genMarkdownBulk(n, "回复")
}

func init() {
	Register(Scenario{
		ID:    "long",
		Build: buildLongSession,
		Live:  false,
	})
}
