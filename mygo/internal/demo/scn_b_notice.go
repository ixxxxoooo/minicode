package demo

import "github.com/ixxxxoooo/minicode/internal/core"

func init() {
	Register(Scenario{
		ID:    "notice-levels",
		Title: "提示 · 三级提示",
		Build: buildNoticeLevels,
		Label: "刚刚",
	})
}

func buildNoticeLevels() []Step {
	s := NewScript("notice-levels", "提示 · 三级提示")
	uid := s.User("把会话流里的 Info / Warning / Error 提示样式各演示几条。")

	s.Notice(core.NoticeInfo, "已切换到 IDE 模式")
	s.Notice(core.NoticeInfo, "上下文已压缩,较早的消息已摘要")
	s.Notice(core.NoticeInfo, "模型已切换为 Sonnet 4.5")

	s.Notice(core.NoticeWarning, "MCP 服务器「github」连接失败,部分工具不可用")
	s.Notice(core.NoticeWarning, "项目未信任,部分写入操作需要额外确认")
	s.Notice(core.NoticeWarning, "磁盘上的 settings.json 已被外部程序修改,已重新加载")

	s.Notice(core.NoticeError, "请求过于频繁(429),请稍后再试")
	s.Notice(core.NoticeError, "鉴权失败:请重新登录 Claude Code(401)")
	s.Notice(core.NoticeError, "网络断开:无法连接 claude 后端,请检查网络")
	s.Notice(core.NoticeError, "上下文已满:请压缩上下文或开启新会话后再试")

	s.Say("以上提示应按级别分列:Info 为灰色行内字,Warning / Error 为带色卡片。")
	s.End(uid, 6, core.StopEndTurn, DefaultStats(1))
	return s.Steps()
}
