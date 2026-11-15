package demo

import (
	"github.com/ixxxxoooo/minicode/internal/core"
)

func init() {
	Register(Scenario{
		ID: "b4-dock-queue", Title: "B4 · 审批排队 1/3", Label: "刚刚",
		Build: buildDockQueue,
	})
	Register(Scenario{
		ID: "b4-queue-stop", Title: "B4 · 排队多条", Label: "刚刚", Live: true,
		Build: buildQueueMulti,
	})
	Register(Scenario{
		ID: "b4-queue-img", Title: "B4 · 排队带图片", Label: "刚刚", Live: true,
		Build: buildQueueMulti,
	})
	Register(Scenario{
		ID: "b4-ask-free", Title: "B4 · 提问自由输入", Label: "刚刚",
		Build: buildAskFree,
	})
	Register(Scenario{
		ID: "b4-user-img", Title: "B4 · 用户消息图片", Label: "刚刚",
		Build: buildUserWithImg,
	})
}

// buildDockQueue 连续三条未处理权限,验证卡片头「1 / 3」。
func buildDockQueue() []Step {
	s := NewScript("b4-dock-queue", "审批排队")
	uid := s.User("请依次处理三个权限")
	s.Say("下面连续出现三条 Bash 权限,卡头应显示 1 / 3。")
	tools := []string{"cargo test router", "cargo fmt --check", "git status -sb"}
	for _, cmd := range tools {
		id := s.ID("p")
		tool := core.NewToolCall(id, "Bash", "运行 "+cmd).WithInput(map[string]any{"command": cmd})
		promotePerm(s, "", id, tool)
	}
	s.Add(ms(200), core.EvNotice{Level: core.NoticeInfo, Text: "请逐条允许;处理完第三条后输入框应淡入并恢复草稿。"})
	return append(s.Steps(), Step{0, core.EvDuration{ID: uid, Secs: 30}})
}

// buildQueueMulti 回合进行中,用于排队卡「停止并发送」演示。
func buildQueueMulti() []Step {
	s := NewScript("b4-queue-stop", "排队演示")
	uid := s.User("假装长跑命令")
	s.Think("长时间运行中…", 8)
	s.Add(ms(200), core.EvTurnStarted{})
	return append(s.Steps(), Step{0, core.EvDuration{ID: uid, Secs: 120}})
}

func buildAskFree() []Step {
	s := NewScript("b4-ask-free", "提问 · 自由输入")
	uid := s.User("开放式问题")
	qid := s.ID("q")
	s.Add(ms(120), core.EvAppend{Item: &core.QuestionRequest{ID: qid, Questions: []core.Question{{
		Question: "部署目标平台是?",
	}}}})
	s.Add(ms(120), core.EvAppend{Item: &core.QuestionRequest{ID: s.ID("q2"), Questions: []core.Question{{
		Question: "若都不合适请说明",
		Options: []core.QuestionOption{
			{Label: "Fly.io", Description: "边缘节点"},
			{Label: "其他", Description: "在下面填写"},
		},
	}}}})
	finishLab(s, uid, 1)
	return s.Steps()
}

func buildUserWithImg() []Step {
	s := NewScript("b4-user-img", "带图片的用户消息")
	uid := s.ID("u")
	s.Add(ms(100), core.EvAppend{Item: &core.User{
		ID: uid, Text: "这是界面里的截图,请描述错误信息",
		Images: []string{"/demo/assets/screenshot-error.png"},
	}})
	s.Say("用户消息已带 Images 字段(会话流里是否展示缩略图见实现状态)。")
	finishLab(s, uid, 1)
	return s.Steps()
}
