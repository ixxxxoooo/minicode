// 演示数据索引 · 自动化
//
// 套 0:默认 4 条(与 pages_test 一致)。
// 套 1:空列表。
// 套 2:20 条挤满周视图,每天/工作日/单周几/每月混合。
// 套 3:多种失败原因(网络超时、权限、超时、未登录)。
// 套 4:一条自动化 60 次运行历史;另有长名称与长提示词。
// 套 5:同一天 09:00 多条重叠。
//
// 切换:命令面板「演示:自动化」。「运行中」需在界面点「立即运行」或命令「演示:自动化·模拟运行中」。
package demo

import (
	"fmt"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/automation"
)

func autoAt(mon time.Time, dayOff int, hm string) time.Time {
	t, _ := time.Parse("15:04", hm)
	d := mon.AddDate(0, 0, dayOff)
	return time.Date(d.Year(), d.Month(), d.Day(), t.Hour(), t.Minute(), 0, 0, time.UTC)
}

func automationsDefault(now time.Time) []automation.Automation {
	return Automations(now)
}

func automationsEmpty() []automation.Automation {
	return nil
}

func automationsCrowded20(now time.Time) []automation.Automation {
	mon := automation.MondayOf(now)
	today := int(now.Weekday()+6) % 7
	days := []string{"每天", "工作日", "周一", "周二", "周三", "周四", "周五", "周六", "周日", "每月 5 日", "每月 15 日", "每月 1 日"}
	var out []automation.Automation
	for i := 1; i <= 20; i++ {
		d := days[i%len(days)]
		h := fmt.Sprintf("%02d:%02d", 7+(i%12), (i*7)%60)
		out = append(out, automation.Automation{
			ID: i, Name: fmt.Sprintf("定时任务 #%02d · %s", i, d), Days: d, Time: h,
			Project: []string{"router-demo", "web-dashboard", "docs-site", "scenario-lab"}[i%4],
			Perm:    "只读 · 计划模式", On: i%5 != 0,
			Prompt: fmt.Sprintf("这是第 %d 条自动化的提示词,用于撑满周历格子。", i),
			Runs:   []automation.Run{{At: autoAt(mon, today-(i%3), h), OK: i%4 != 0}},
		})
	}
	return out
}

func automationsFailures(now time.Time) []automation.Automation {
	mon := automation.MondayOf(now)
	today := int(now.Weekday()+6) % 7
	errs := []string{"网络超时", "权限被拒:无法写入仓库", "运行超时(>10 分钟)", "claude 未登录:请运行 claude login"}
	var out []automation.Automation
	for i, e := range errs {
		id := i + 1
		out = append(out, automation.Automation{
			ID: id, Name: "失败样例 · " + e, Days: "每天", Time: fmt.Sprintf("%02d:15", 9+i),
			Project: "router-demo", Perm: "只读 · 计划模式", On: true,
			Prompt: "故意失败的演示任务。",
			Runs:   []automation.Run{{At: autoAt(mon, today, fmt.Sprintf("%02d:15", 9+i)), Err: e}},
		})
	}
	out = append(out, automation.Automation{
		ID: 5, Name: "已停用 · 从不运行", Days: "周三", Time: "11:00", Project: "docs-site", Perm: "只读 · 计划模式", On: false,
		Prompt: "暂停中。", Runs: nil,
	})
	out = append(out, automation.Automation{
		ID: 6, Name: "从未运行", Days: "周五", Time: "16:00", Project: "web-dashboard", Perm: "只读 · 计划模式", On: true,
		Prompt: "新建后还没触发过。", Runs: nil,
	})
	nextToday := autoAt(mon, today, "18:30")
	nextWeek := autoAt(mon, today+4, "09:00")
	if today > 3 {
		nextWeek = autoAt(mon, today-3, "09:00")
	}
	out = append(out,
		automation.Automation{ID: 7, Name: "下次 · 今天傍晚", Days: weekdaysFor(now), Time: "18:30", Project: "router-demo", Perm: "只读 · 计划模式", On: true,
			Prompt: "下次运行在今天。", Runs: []automation.Run{{At: nextToday.Add(-24 * time.Hour), OK: true}}},
		automation.Automation{ID: 8, Name: "下次 · 跨周", Days: "周一", Time: "09:00", Project: "router-demo", Perm: "只读 · 计划模式", On: true,
			Prompt: "下次在下周一。", Runs: []automation.Run{{At: nextWeek, OK: true}}},
	)
	return out
}

func weekdaysFor(now time.Time) string {
	wd := int(now.Weekday()+6) % 7
	if wd < 5 {
		return "工作日"
	}
	return automation.Weekdays()[wd]
}

func automationsLongHistory(now time.Time) []automation.Automation {
	mon := automation.MondayOf(now)
	var runs []automation.Run
	for i := 0; i < 60; i++ {
		runs = append(runs, automation.Run{At: mon.Add(-time.Duration(i) * 6 * time.Hour), OK: i%9 != 0, Err: map[bool]string{true: "", false: "偶发失败"}[i%9 != 0]})
	}
	name := strings.Repeat("超长名称·", 6) + "巡检全仓库依赖与许可证合规性"
	prompt := strings.Repeat("请逐项目检查 Cargo/npm 依赖是否有 CVE;输出表格。 ", 8)
	return []automation.Automation{{
		ID: 1, Name: name, Days: "每天", Time: "06:00", Project: "scenario-lab", Perm: "只读 · 计划模式", On: true,
		Prompt: prompt, Runs: runs,
	}}
}

func automationsOverlap(now time.Time) []automation.Automation {
	mon := automation.MondayOf(now)
	today := int(now.Weekday()+6) % 7
	var out []automation.Automation
	for i := 1; i <= 5; i++ {
		out = append(out, automation.Automation{
			ID: i, Name: fmt.Sprintf("重叠 09:00 · 任务 %d", i), Days: "每天", Time: "09:00",
			Project: "router-demo", Perm: "只读 · 计划模式", On: true,
			Prompt: "同一时间触发,周视图应叠放或多行。",
			Runs:   []automation.Run{{At: autoAt(mon, today, "09:00"), OK: true}},
		})
	}
	return out
}

// AutomationsFor 返回指定套的自动化清单。
func AutomationsFor(set int, now time.Time) []automation.Automation {
	switch set {
	case 1:
		return automationsEmpty()
	case 2:
		return automationsCrowded20(now)
	case 3:
		return automationsFailures(now)
	case 4:
		return automationsLongHistory(now)
	case 5:
		return automationsOverlap(now)
	default:
		return automationsDefault(now)
	}
}
