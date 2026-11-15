package demo

import (
	"fmt"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func init() {
	Register(Scenario{ID: "c-cards-chg", Title: "待处理 · Changes 汇总卡", Label: "刚刚", Build: buildChanges})
	Register(Scenario{ID: "c-cards-mix", Title: "待处理 · 画布回退与模式提示", Label: "刚刚", Build: buildCardsMix})
}

func buildChanges() []Step {
	s := NewScript("c-cards-chg", "待处理 · Changes 汇总卡")
	uid := s.User("改动汇总")
	files := []core.ChangedFile{
		{Path: "src/router.rs", Added: 12, Removed: 3},
		{Path: "src/handlers/mod.rs", Added: 0, Removed: 0},
		{Path: "tests/router_test.rs", Added: 28, Removed: 0},
	}
	for i := 0; i < 12; i++ {
		files = append(files, core.ChangedFile{Path: fmt.Sprintf("src/module_%02d.rs", i), Added: i + 1, Removed: i % 3})
	}
	s.Add(ms(120), core.EvAppend{Item: &core.Changes{ID: s.ID("ch"), Files: files}})
	s.Say("含多文件、0 增删与大量文件的 Changes 卡。")
	finishLab(s, uid, 32)
	return s.Steps()
}

func buildCardsMix() []Step {
	s := NewScript("c-cards-mix", "待处理 · 画布回退与模式提示")
	uid := s.User("其他卡片")
	s.Add(ms(100), core.EvAppend{Item: &core.Canvas{ID: s.ID("cv"), Title: "路由架构草图"}})
	s.Add(ms(80), core.EvAppend{Item: &core.Canvas{ID: s.ID("cv"), Title: "错误处理时序图"}})
	s.Add(ms(80), core.EvAppend{Item: &core.Canvas{ID: s.ID("cv"), Title: "部署拓扑"}})

	const rwUUID = "demo-rewind-uuid"
	s.Add(ms(120), core.EvUserUUID{ID: uid, UUID: rwUUID})
	s.Add(ms(80), core.EvRewind{Outcome: core.RewindOutcome{
		UUID: rwUUID, DryRun: true, CanRewind: true,
		Files: []string{"src/router.rs", "tests/router_test.rs"}, Insertions: 18, Deletions: 4,
	}})
	s.Add(ms(80), core.EvRewind{Outcome: core.RewindFailed(rwUUID, false, "No file checkpoint found for this message.")})
	s.Add(0, core.EvNotice{Level: core.NoticeInfo, Text: "回退预览:将恢复 2 个文件(+18 / -4 行)。需从用户消息点「回退文件」才会弹出确认卡。"})

	s.Add(0, core.EvNotice{Level: core.NoticeInfo, Text: "已切换到接受编辑模式"})
	s.Add(0, core.EvModeChanged{Mode: core.ModeAcceptEdits})
	s.Add(0, core.EvNotice{Level: core.NoticeInfo, Text: "模型已切换为 Sonnet 5.5"})
	s.Add(0, core.EvModelChanged{Model: "claude-sonnet-5-5"})
	finishLab(s, uid, 33)
	return s.Steps()
}
