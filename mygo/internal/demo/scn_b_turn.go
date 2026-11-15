package demo

import (
	"strings"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func init() {
	Register(Scenario{
		ID:    "turn-endings",
		Title: "回合 · 收尾与统计",
		Build: buildTurnEndings,
		Label: "刚刚",
	})
}

func buildTurnEndings() []Step {
	s := NewScript("turn-endings", "回合 · 收尾与统计")

	// 正常 + 完整统计
	uid := s.User("正常结束:展示完整回合统计")
	s.Say("这是正常结束的助手回复,下面应有模型名、用时与 token 明细。")
	s.End(uid, 73, core.StopEndTurn, DefaultStats(10))

	// 无费用
	uid = s.User("第三方服务商:无费用字段")
	s.Say("费用项应不出现。")
	s.End(uid, 12, core.StopEndTurn, statsNoCost())

	// 无缓存
	uid = s.User("无缓存命中")
	s.Say("操作条不应显示缓存命中率。")
	s.End(uid, 9, core.StopEndTurn, statsNoCache())

	// 超大数字
	uid = s.User("超大 token 与费用数字")
	s.Say("检查 ↑ ↓ 与费用的缩写格式。")
	s.End(uid, 120, core.StopEndTurn, statsHuge())

	// 极短用时
	uid = s.User("极短用时 1 秒")
	s.Say("好。")
	s.End(uid, 1, core.StopEndTurn, statsTiny())

	// 极长用时 3 小时
	uid = s.User("极长用时 3 小时")
	s.Say("长任务演示。")
	s.End(uid, 3*3600, core.StopEndTurn, DefaultStats(11))

	// 被中断(会由状态机追加「已中断」)
	uid = s.User("请在中途取消本回合")
	aid := s.ID("a")
	s.Add(ms(200), core.EvAppend{Item: &core.Assistant{ID: aid, Streaming: true}})
	s.Add(ms(80), core.EvTextDelta{ID: aid, Delta: "正在输出,即将被打断…"})
	s.End(uid, 8, core.StopCancelled, DefaultStats(12))

	// 拒绝
	uid = s.User("请帮我绕过付费墙下载受版权保护的全套电影")
	s.Say("抱歉,我不能协助侵犯版权或绕过付费限制的请求。你可以改用合法的订阅或购买渠道。")
	s.End(uid, 4, core.StopRefusal, DefaultStats(13))

	// 超出最大长度
	trunc := strings.Repeat("这是一段很长的回复,用于演示输出上限截断.", 80)
	uid = s.User("写一份超长文档")
	s.Say(trunc + "\n\n(输出已达上限,后续内容被截断)")
	s.End(uid, 55, core.StopMaxTokens, statsHuge())

	// 请求失败 + 错误提示
	uid = s.User("模拟限流失败")
	s.Notice(core.NoticeError, "上游限流(429):请在 60 秒后重试,或切换到用量更低的模型。")
	s.End(uid, 2, core.StopError, statsTiny())

	// 连续第二回合(多回合衔接)
	uid = s.User("上一回合失败后继续")
	s.Say("新问题,新的操作条。")
	s.End(uid, 7, core.StopEndTurn, DefaultStats(14))

	// 只有工具,无助手正文
	uid = s.User("只跑命令,不用文字总结")
	s.Think("确认测试套件名称。", 2)
	tool := core.NewToolCall("", "Bash", "运行测试").WithInput(map[string]any{"command": "cargo test router"})
	s.Tool(tool, core.ToolCompleted, 1800, &core.TerminalBlock{Command: "cargo test router", Output: "test result: ok. 12 passed", ExitCode: core.Int(0)})
	s.End(uid, 9, core.StopEndTurn, DefaultStats(15))

	// 只有思考,无正文
	uid = s.User("先想方案,这轮不要正文")
	s.Think("权衡三种实现:1) 归一化路径 2) 大小写不敏感比较 3) 配置开关。选 1 改动最小。", 11)
	s.End(uid, 14, core.StopEndTurn, DefaultStats(16))

	// 空正文(界面不渲染空 Assistant)
	uid = s.User("本轮回复留空")
	s.Say("   ")
	s.End(uid, 3, core.StopEndTurn, statsTiny())

	return s.Steps()
}
