package demo

import (
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func init() {
	Register(Scenario{
		ID:    "flow-gaps-b4",
		Title: "会话流 · B4 缺口合集",
		Label: "刚刚",
		Build: buildFlowGapsB4,
		Group: "scenario-lab",
	})
}

func flowGapAsset(name string) string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return name
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "..")
	return filepath.Join(root, "assets", "icon", name)
}

func buildFlowGapsB4() []Step {
	s := NewScript("flow-gaps-b4", "会话流 · B4 缺口合集")

	// 1) 后台任务通知行(正常 / 停止 / 监听)
	uid := s.User("演示 TaskNote、并行子 Agent、回合折叠、提示动作与用户消息附件。")
	t1 := core.NewBackgroundTask(time.Now().UnixMilli()-120_000, s.ID("sh"), core.TaskShell, "cargo watch")
	t1.OutputFile = "target/dev.log"
	s.Add(ms(50), core.EvTaskStarted{Task: t1})
	s.Add(ms(80), core.EvTaskNotification{TaskID: t1.TaskID, Status: core.TaskCompleted, Summary: "后台命令 cargo watch 已结束(退出码 0)"})
	s.Add(ms(40), core.EvAppend{Item: &core.TaskNote{ID: s.ID("n1"), TaskID: t1.TaskID, Kind: core.TaskShell, Status: core.TaskCompleted, Text: "后台命令 cargo watch 已结束(退出码 0)"}})

	t2 := core.NewBackgroundTask(time.Now().UnixMilli()-90_000, s.ID("npm"), core.TaskShell, "npm run dev")
	s.Add(ms(40), core.EvTaskNotification{TaskID: t2.TaskID, Status: core.TaskStopped, Summary: "后台命令 npm run dev 已停止"})
	s.Add(ms(40), core.EvAppend{Item: &core.TaskNote{ID: s.ID("n2"), TaskID: t2.TaskID, Kind: core.TaskShell, Status: core.TaskStopped, Text: "后台命令 npm run dev 已停止"}})

	s.Add(ms(40), core.EvAppend{Item: &core.TaskNote{ID: s.ID("n3"), TaskID: "mon1", Kind: core.TaskMonitor, Status: core.TaskCompleted, Text: "监听 target/dev.log 触发了 3 次"}})

	// 2) 并行子 Agent
	for i := 0; i < 2; i++ {
		c := core.NewSubagent(s.ID("p"), "code-reviewer", fmt.Sprintf("审查 handlers #%d", i+1), core.SubRunning)
		stampSub(c, time.Duration(i+1)*15*time.Second)
		s.Add(ms(30), core.EvAppend{Item: c})
	}

	// 3) 回合步骤(结束后应折成「已处理」)
	rid := s.ID("r")
	s.Add(ms(50), core.EvAppend{Item: core.NewToolCall(rid, "Read", "读取 router.rs").WithInput(map[string]any{"file_path": "src/router.rs"})})
	s.Add(ms(60), core.EvToolUpdate{ID: rid, Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), AppendContent: []core.ToolContent{&core.TextBlock{Text: "86 行"}}}})
	eid := s.ID("e")
	s.Add(ms(50), core.EvAppend{Item: core.NewToolCall(eid, "Edit", "编辑 router.rs").WithInput(map[string]any{"file_path": "src/router.rs"})})
	s.Add(ms(80), core.EvToolUpdate{ID: eid, Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), AppendContent: []core.ToolContent{
		&core.DiffBlock{Path: "src/router.rs", Old: "a", New: "b", Line: 4},
	}}})

	// 4) 提示分级(含压缩 / 错误重试)
	s.Add(ms(40), core.EvNotice{Level: core.NoticeInfo, Text: "已切换到「计划模式」"})
	s.Add(ms(40), core.EvNotice{Level: core.NoticeWarning, Text: "上下文已用 92%,建议压缩后继续。"})
	s.Add(ms(40), core.EvNotice{Level: core.NoticeError, Text: "请求失败 · 529 服务过载"})

	// 5) 权限拒绝专用行
	bid := s.ID("deny")
	bash := core.NewToolCall(bid, "Bash", "rm -rf target").WithInput(map[string]any{"command": "rm -rf target"})
	promotePerm(s, "", bid, bash)
	resolvePerm(s, "", bid, "deny", false)

	s.Say("上方应出现并行 2 个子 Agent 汇总行;回合结束后步骤应收成「已处理 …」。")
	finishLab(s, uid, 80)

	// 6) 用户消息图片 + 回退入口(第二条回合)
	img := flowGapAsset("tray-error.png")
	img2 := flowGapAsset("tray-idle.png")
	uid2 := s.ID("u2")
	s.Add(0, core.EvUserUUID{ID: uid2, UUID: "flow-gap-rewind-uuid"})
	s.Add(0, core.EvAppend{Item: &core.User{
		ID: uid2, Text: "附带两张截图与一个找不到的文件;点缩略图应打开灯箱,点卡片应悬浮编辑。",
		Images: []string{img, img2, "/demo/assets/missing-screenshot.png"},
	}})
	s.Add(ms(200), core.EvTurnStarted{})
	s.Say("点击缩略图可在会话流内展开预览;有 uuid 时可点「回退文件」走确认卡。")
	s.End(uid2, 4, core.StopEndTurn, DefaultStats(10))

	// 7) 失败回合保持展开(不折叠)
	uid3 := s.User("失败回合不应折叠步骤")
	fid := s.ID("fail")
	s.Add(ms(40), core.EvAppend{Item: core.NewToolCall(fid, "Bash", "cargo build").WithInput(map[string]any{"command": "cargo build"})})
	s.Add(ms(200), core.EvToolUpdate{ID: fid, Patch: core.ToolPatch{Status: core.Status(core.ToolFailed), AppendContent: []core.ToolContent{
		&core.TerminalBlock{Command: "cargo build", Output: "error: could not compile", ExitCode: core.Int(101)},
	}}})
	s.Add(ms(40), core.EvNotice{Level: core.NoticeError, Text: "命令退出码 101"})
	s.Say("本回合有失败步骤,应始终展开步骤列表。")
	s.End(uid3, 6, core.StopError, DefaultStats(11))

	return s.Steps()
}
