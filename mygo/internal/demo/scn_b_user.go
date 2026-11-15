package demo

import (
	"strings"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func init() {
	Register(Scenario{
		ID:    "msg-variants",
		Title: "消息 · 用户消息变体",
		Build: buildUserVariants,
		Label: "刚刚",
	})
}

func buildUserVariants() []Step {
	s := NewScript("msg-variants", "消息 · 用户消息变体")

	// 短句
	uid := s.User("好,就按这个方案做。")
	s.Say("收到。")
	s.End(uid, 2, core.StopEndTurn, statsTiny())

	// 多行
	uid = s.User("分三步:\n1. 读 router.rs\n2. 加 normalize\n3. 补测试")
	s.Say("明白,按步骤来。")
	s.End(uid, 3, core.StopEndTurn, DefaultStats(1))

	// 超长(>6 行折叠)
	longList := strings.Join([]string{
		"请按下面的清单逐项处理:",
		"1. 把 normalize 抽到单独的模块",
		"2. 补 4 条大小写用例",
		"3. 修复 CI 里失败的 clippy 警告",
		"4. 更新 README 的路由示例",
		"5. 把 fallback 逻辑改成显式配置",
		"6. 清理未使用的依赖",
		"7. 补充发布说明",
		"8. 对照设计稿截图",
	}, "\n")
	uid = s.User(longList)
	s.Say("清单收到,我会逐项处理。")
	s.End(uid, 5, core.StopEndTurn, DefaultStats(2))

	// 超长单行 curl
	curl := "curl -X POST 'https://api.example.com/v1/chat/completions' -H 'Authorization: Bearer sk-••••' -H 'Content-Type: application/json' -d '{\"model\":\"demo\",\"messages\":[{\"role\":\"user\",\"content\":\"hello\"}]}'"
	uid = s.User(curl)
	s.Say("命令已记录(演示数据中的密钥已遮盖)。")
	s.End(uid, 1, core.StopEndTurn, statsTiny())

	// 多个 @ 文件与深路径
	fileMsg := "请同时看 @{src/router.rs}、@{docs/api/guide.md} 和 @{packages/ui/src/components/forms/DatePicker.tsx}"
	uid = s.User(fileMsg, "src/router.rs", "docs/api/guide.md", "packages/ui/src/components/forms/DatePicker.tsx")
	s.Say("已加载三处引用。")
	s.End(uid, 4, core.StopEndTurn, DefaultStats(4))

	// 图片附件(数据在 Images;界面是否渲染见汇报)
	uid = s.ID("u")
	s.Add(0, core.EvAppend{Item: &core.User{
		ID: uid, Text: "这张报错截图里红框标出的栈顶是什么?",
		Images: []string{"/demo/assets/screenshot-error.png"},
	}})
	s.Add(ms(250), core.EvTurnStarted{})
	s.Say("截图里的栈顶是 `normalize` 未定义(演示回复)。")
	s.End(uid, 6, core.StopEndTurn, DefaultStats(5))

	// 反引号与围栏代码混在用户消息里
	uid = s.User("把 `match_path` 改成:\n\n```rust\nself.routes.get(&key)\n```\n\n不要动测试。")
	s.Say("好的,只改匹配逻辑。")
	s.End(uid, 3, core.StopEndTurn, DefaultStats(6))

	// CJK + emoji
	uid = s.User("今天把会话流样式过一遍 🎯,中文 English 混排也要对齐设计稿 ✨")
	s.Say("没问题。")
	s.End(uid, 2, core.StopEndTurn, statsTiny())

	// 斜杠命令样式
	uid = s.User("/release 2.1.0\n请生成发布说明并打 tag")
	s.Say("将按自定义命令模板生成发布说明(演示)。")
	s.End(uid, 8, core.StopEndTurn, DefaultStats(8))

	// 「编辑过」:核心无独立标记;先结束一轮,再以编辑后的全文重开一轮(对标编辑并重新发送)
	uid = s.User("先只改 router.rs")
	s.Say("好,只动 router.rs。")
	s.End(uid, 2, core.StopEndTurn, statsTiny())
	uid = s.User("更正:router.rs 与 tests/router_test.rs 一起改,并跑 cargo test")
	s.Say("收到更正,会同时改两处并跑测试。")
	s.End(uid, 4, core.StopEndTurn, DefaultStats(9))

	return s.Steps()
}
