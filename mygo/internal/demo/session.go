package demo

import (
	"github.com/ixxxxoooo/minicode/internal/core"
)

// SessionTitle 按 id 找侧栏里的会话标题;找不到返回空串。
func SessionTitle(id string) string {
	for _, p := range Projects() {
		for _, s := range p.Sessions {
			if s.ID == id {
				return s.Title
			}
		}
	}
	return ""
}

// Session 为侧栏里的某个会话造后端:
//   - 登记在 registry.go 里的场景(Live 按节奏播放,否则一次性回放);
//   - fixbug:历史会话,一次性回放完整回合;
//   - perm:启动后自动发一条会触发权限请求的提示,停在权限卡上等用户处理;
//   - 空 id:全新会话(欢迎页),没有任何条目;
//   - 其余:只有标题的空会话,但发消息照样能得到回复。
func Session(id string, speed float64) *Backend {
	ready := func(title string) []Step {
		s := []Step{{0, core.EvReady{SessionID: "demo-" + id, Models: core.DefaultModels(), Model: "sonnet", Mode: core.ModeDefault}}}
		if title != "" {
			s = append(s, Step{0, core.EvTitleChanged{Title: title}})
		}
		return s
	}
	// 先查场景注册表(scn_*.go 登记的各种样式场景)
	if sc, ok := findScenario(id); ok && sc.Build != nil {
		b := New(sc.Build, speed)
		b.Instant = !sc.Live
		b.Auto = sc.Auto
		return b
	}
	switch id {
	case "fixbug":
		b := New(func() []Step { return FixBug(false) }, speed)
		b.Instant = true
		return b
	case "perm":
		b := New(func() []Step { return ready(SessionTitle(id)) }, speed)
		b.Instant = true
		b.Auto = "运行 cargo test router,看看权限请求怎么弹"
		return b
	}
	b := New(func() []Step { return ready(SessionTitle(id)) }, speed)
	b.Instant = true
	return b
}
