package demo

import (
	"time"

	"github.com/ixxxxoooo/minicode/internal/automation"
)

// Automations 是演示用的自动化清单。运行记录按 now 所在周相对地造出,所以随时跑都和设计稿一个样子。
func Automations(now time.Time) []automation.Automation {
	mon := automation.MondayOf(now)
	at := func(dayOff int, hm string) time.Time {
		t, _ := time.Parse("15:04", hm)
		d := mon.AddDate(0, 0, dayOff)
		return time.Date(d.Year(), d.Month(), d.Day(), t.Hour(), t.Minute(), 0, 0, time.UTC)
	}
	today := int(now.Weekday()+6) % 7
	return []automation.Automation{
		{ID: 1, Name: "日报摘要", Days: "工作日", Time: "09:00", Project: "router-demo", Perm: "只读 · 计划模式", On: true,
			Prompt: "汇总昨天的提交与未关闭的 PR,生成一份简短的日报,按「已完成 / 进行中 / 阻塞」分组。",
			Runs: []automation.Run{
				{At: at(today, "09:00"), OK: true, SessionID: AutomationDemoSessionID, Key: "auto:1:ok"},
				{At: at(today-1, "09:00"), OK: false, Err: "已取消", SessionID: AutomationDemoSessionID, Output: "[演示] 用户取消运行\n", Key: "auto:1:cancel"},
				{At: at(today-2, "09:00"), OK: true, SessionID: AutomationDemoSessionID},
				{At: at(today-3, "09:00"), OK: true}, // 旧记录无 session_id
			}},
		{ID: 2, Name: "依赖升级检查", Days: "周二", Time: "14:00", Project: "web-dashboard", Perm: "只读 · 计划模式", On: true,
			Prompt: "检查 Cargo.toml 里可升级的依赖,列出破坏性变更,并给出升级顺序建议。不要修改任何文件。",
			Runs: []automation.Run{
				{At: at(1, "14:00"), Err: "网络超时", Output: "错误:网络超时(演示)\n"},
				{At: at(-6, "14:00"), OK: true, SessionID: AutomationDemoSessionID},
				{At: at(-13, "14:00"), OK: true},
			}},
		{ID: 3, Name: "周报生成", Days: "周五", Time: "17:30", Project: "router-demo", Perm: "只读 · 计划模式", On: true,
			Prompt: "根据本周所有已合并的提交生成周报,突出对用户可见的变化。",
			Runs:   []automation.Run{{At: at(4, "17:30"), OK: true}, {At: at(-3, "17:30"), OK: true}}},
		{ID: 4, Name: "备份检查", Days: "每天", Time: "20:00", Project: "docs-site", Perm: "只读 · 计划模式", On: false,
			Prompt: "确认 docs-site 的备份任务今天已完成,没有则提醒。",
			Runs:   []automation.Run{{At: at(-1, "20:00"), OK: true}}},
	}
}
