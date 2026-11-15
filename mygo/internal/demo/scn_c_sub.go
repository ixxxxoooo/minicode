package demo

import (
	"fmt"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func init() {
	Register(Scenario{
		ID: "subagent", Project: "web-dashboard", Title: "子 Agent · 代码审查与调研全集", Label: "昨天", Group: "昨天",
		Build: buildSubGallery,
	})
	Register(Scenario{ID: "c-sub-run", Title: "子 Agent · 运行中嵌套时间线", Label: "刚刚", Live: true, Build: buildSubRunning})
	Register(Scenario{ID: "c-sub-bg", Title: "子 Agent · 后台运行", Label: "刚刚", Build: buildSubBackground})
	Register(Scenario{ID: "c-sub-done", Title: "子 Agent · 完成", Label: "刚刚", Build: buildSubDone})
	Register(Scenario{ID: "c-sub-fail", Title: "子 Agent · 失败默认展开", Label: "刚刚", Build: buildSubFail})
	Register(Scenario{ID: "c-sub-stop", Title: "子 Agent · 被停止", Label: "刚刚", Build: buildSubStopped})
	Register(Scenario{ID: "c-sub-stall", Title: "子 Agent · 停滞", Label: "刚刚", Build: buildSubStall})
	Register(Scenario{ID: "c-sub-nest", Title: "子 Agent · 嵌套子 Agent", Label: "刚刚", Build: buildSubNest})
	Register(Scenario{ID: "c-sub-parallel", Title: "子 Agent · 并行多个", Label: "刚刚", Build: buildSubParallel})
	Register(Scenario{ID: "c-sub-heavy", Title: "子 Agent · 高统计长摘要", Label: "刚刚", Build: buildSubHeavy})
}

func buildSubGallery() []Step {
	s := NewScript("subagent", "子 Agent · 代码审查与调研全集")
	uid := s.User("子 Agent 各状态一览")
	s.Say("本页汇总运行中 / 后台 / 完成 / 失败 / 停止 / 停滞,以及嵌套时间线与并行派发。")

	// 完成
	done := core.NewSubagent(s.ID("d"), "code-reviewer", "审查 handlers 错误处理", core.SubCompleted)
	done.Summary = "未发现 panic 路径,建议把 unwrap 换成 ?"
	done.Tokens, done.ToolCount = 4200, 6
	stampSub(done, time.Minute)
	s.Add(0, core.EvAppend{Item: done})

	// 失败
	fail := core.NewSubagent(s.ID("f"), "Explore", "调研缓存方案", core.SubFailed)
	fail.Summary = strings.Repeat("连接 Redis 超时,已重试 3 次。", 8)
	fail.Tokens, fail.ToolCount = 1800, 4
	stampSub(fail, 2*time.Minute)
	s.Add(0, core.EvAppend{Item: fail})

	// 后台 + task 同步
	bg := core.NewSubagent(s.ID("b"), "general-purpose", "后台整理导入", core.SubBackground)
	bg.TaskID = "sub-bg1"
	bg.Summary = "正在重排 use 语句…"
	stampSub(bg, 20*time.Second)
	s.Add(0, core.EvAppend{Item: bg})
	t := core.NewBackgroundTask(time.Now().UnixMilli()-15_000, "sub-bg1", core.TaskAgent, "后台整理导入")
	t.ToolUseID, t.Backgrounded, t.Summary = bg.ID, true, bg.Summary
	s.Add(0, core.EvTaskStarted{Task: t})

	// 停滞
	stall := core.NewSubagent(s.ID("s"), "Explore", "抓取第三方 API 文档", core.SubRunning)
	stall.LastActivityMs = time.Now().UnixMilli() - core.StallMs - 5000
	stall.StartedMs = stall.LastActivityMs - 120_000
	s.Add(0, core.EvAppend{Item: stall})

	// 并行三个
	for i := 0; i < 3; i++ {
		p := core.NewSubagent(s.ID("p"), "code-reviewer", fmt.Sprintf("并行审查 pkg::mod%d", i+1), core.SubRunning)
		stampSub(p, time.Duration(i+1)*3*time.Second)
		s.Add(0, core.EvAppend{Item: p})
	}

	s.Say("点卡片可在工作区打开详情;失败卡应默认展开摘要。")
	finishLab(s, uid, 20)
	return s.Steps()
}

func nestTools(s *Script, parent string) {
	rid := s.ID("r")
	s.Add(ms(100), core.EvAppend{Parent: parent, Item: core.NewToolCall(rid, "Read", "读取 lib.rs").WithInput(map[string]any{"file_path": "src/lib.rs"})})
	s.Add(ms(100), core.EvToolUpdate{Parent: parent, ID: rid, Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), AppendContent: []core.ToolContent{&core.TextBlock{Text: "86 lines"}}}})
	gid := s.ID("g")
	s.Add(ms(80), core.EvAppend{Parent: parent, Item: core.NewToolCall(gid, "Grep", "搜索 TODO").WithInput(map[string]any{"pattern": "TODO", "path": "src/"})})
	s.Add(ms(200), core.EvToolUpdate{Parent: parent, ID: gid, Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), AppendContent: []core.ToolContent{&core.TextBlock{Text: "2 matches"}}}})
	s.Add(ms(80), core.EvAppend{Parent: parent, Item: &core.Thinking{ID: s.ID("t"), Text: "考虑把错误统一成 AppError。"}})
	edit := core.NewToolCall(s.ID("e"), "Edit", "编辑 lib.rs").WithInput(map[string]any{"file_path": "src/lib.rs"})
	s.Add(ms(80), core.EvAppend{Parent: parent, Item: edit})
	s.Add(ms(200), core.EvToolUpdate{Parent: parent, ID: edit.ID, Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), AppendContent: []core.ToolContent{
		&core.DiffBlock{Path: "src/lib.rs", Old: "unwrap()", New: "?;", Line: 12},
	}}})
}

func buildSubRunning() []Step {
	s := NewScript("c-sub-run", "子 Agent · 运行中嵌套时间线")
	uid := s.User("运行中")
	card := core.NewSubagent(s.ID("a"), "code-reviewer", "审查 PR #128", core.SubRunning)
	stampSub(card, 5*time.Second)
	s.Add(ms(120), core.EvAppend{Item: card})
	nestTools(s, card.ID)
	return append(s.Steps(), Step{0, core.EvDuration{ID: uid, Secs: 8}})
}

func buildSubBackground() []Step {
	s := NewScript("c-sub-bg", "子 Agent · 后台运行")
	uid := s.User("后台")
	c := core.NewSubagent(s.ID("a"), "general-purpose", "整理文档链接", core.SubBackground)
	c.TaskID = "csub-bg"
	stampSub(c, time.Minute)
	s.Add(0, core.EvAppend{Item: c})
	t := core.NewBackgroundTask(time.Now().UnixMilli()-40_000, "csub-bg", core.TaskAgent, c.Description)
	t.ToolUseID, t.Backgrounded = c.ID, true
	s.Add(0, core.EvTaskStarted{Task: t})
	s.Add(ms(200), core.EvTaskProgress{TaskID: "csub-bg", Summary: "已处理 12/40 个链接"})
	finishLab(s, uid, 21)
	return s.Steps()
}

func buildSubDone() []Step {
	s := NewScript("c-sub-done", "子 Agent · 完成")
	uid := s.User("完成")
	id := s.ID("a")
	s.Add(0, core.EvAppend{Item: core.NewSubagent(id, "Explore", "对比 axum / actix 路由", core.SubRunning)})
	s.Add(ms(300), core.EvSubagentFinished{ID: id, Status: core.SubCompleted, Summary: "axum 路由更偏类型安全,actix 生态更广。", Tokens: 6200, ToolCount: 9, DurationMs: 95_000})
	finishLab(s, uid, 22)
	return s.Steps()
}

func buildSubFail() []Step {
	s := NewScript("c-sub-fail", "子 Agent · 失败默认展开")
	uid := s.User("失败")
	id := s.ID("a")
	s.Add(0, core.EvAppend{Item: core.NewSubagent(id, "code-reviewer", "审查 unsafe 块", core.SubRunning)})
	s.Add(ms(400), core.EvSubagentFinished{ID: id, Status: core.SubFailed, Summary: "无法解析 build.rs 里的 cfg 分支,审查中止。", Tokens: 900, ToolCount: 2, DurationMs: 12_000})
	finishLab(s, uid, 23)
	return s.Steps()
}

func buildSubStopped() []Step {
	s := NewScript("c-sub-stop", "子 Agent · 被停止")
	uid := s.User("停止")
	c := core.NewSubagent(s.ID("a"), "Explore", "抓取多个镜像站", core.SubStopped)
	c.Summary = "用户停止了任务"
	stampSub(c, time.Minute)
	s.Add(0, core.EvAppend{Item: c})
	finishLab(s, uid, 24)
	return s.Steps()
}

func buildSubStall() []Step {
	s := NewScript("c-sub-stall", "子 Agent · 停滞")
	uid := s.User("停滞")
	c := core.NewSubagent(s.ID("a"), "Explore", "等待沙盒 API", core.SubRunning)
	c.StartedMs = time.Now().UnixMilli() - 180_000
	c.LastActivityMs = time.Now().UnixMilli() - core.StallMs - 10_000
	s.Add(0, core.EvAppend{Item: c})
	finishLab(s, uid, 25)
	return s.Steps()
}

func buildSubNest() []Step {
	s := NewScript("c-sub-nest", "子 Agent · 嵌套子 Agent")
	uid := s.User("嵌套")
	outer := core.NewSubagent(s.ID("o"), "general-purpose", "总控:拆分调研", core.SubRunning)
	stampSub(outer, 10*time.Second)
	s.Add(0, core.EvAppend{Item: outer})
	inner := core.NewSubagent(s.ID("i"), "Explore", "子任务:读 release note", core.SubCompleted)
	inner.Summary = "2.0 弃用了旧中间件"
	stampSub(inner, 8*time.Second)
	s.Add(0, core.EvAppend{Parent: outer.ID, Item: inner})
	nestTools(s, outer.ID)
	finishLab(s, uid, 26)
	return s.Steps()
}

func buildSubParallel() []Step {
	s := NewScript("c-sub-parallel", "子 Agent · 并行多个")
	uid := s.User("并行")
	for i := 0; i < 4; i++ {
		c := core.NewSubagent(s.ID("p"), "code-reviewer", fmt.Sprintf("审查模块 %d", i+1), core.SubRunning)
		stampSub(c, time.Duration(i)*2*time.Second)
		s.Add(ms(50), core.EvAppend{Item: c})
	}
	s.Say("连续多张运行中卡片用于验证「并行 N 个」汇总(若界面已实现)。")
	finishLab(s, uid, 27)
	return s.Steps()
}

func buildSubHeavy() []Step {
	s := NewScript("c-sub-heavy", "子 Agent · 高统计长摘要")
	uid := s.User("高统计")
	c := core.NewSubagent(s.ID("a"), "code-reviewer", "全仓静态扫描", core.SubCompleted)
	c.Summary = strings.Repeat("扫描 1.2k 文件,发现 4 处 unwrap 与 2 处 expect,建议统一错误类型并补 6 条测试。", 3)
	c.Tokens, c.ToolCount = 128_000, 84
	stampSub(c, 8*time.Minute)
	s.Add(0, core.EvAppend{Item: c})
	finishLab(s, uid, 28)
	return s.Steps()
}
