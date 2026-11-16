package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

var (
	morphFadeOut     = kit.Scale(120 * time.Millisecond)
	morphFadeInDelay = kit.Scale(90 * time.Millisecond)
	morphWsInDelay   = kit.Scale(60 * time.Millisecond) // IDE→Agent 时工作区回到右侧
)

// morphOpacity 按设计稿 forms 页各轨道时间轴算内容透明度(动效关闭或已结束时恒为 1)。
func (a *App) morphOpacity(track string) float32 {
	if !kit.MotionOn() || morph.Duration == 0 || a.morphAt.IsZero() || !a.morphing() {
		return 1
	}
	el := time.Since(a.morphAt)
	toIDE := a.mode == ModeIDE && a.morphFrom == ModeAgent
	toAgent := a.mode == ModeAgent && a.morphFrom == ModeIDE
	switch track {
	case "ws":
		if toIDE {
			if el < morphFadeInDelay {
				return 0
			}
			p := float32(el-morphFadeInDelay) / float32(max(time.Millisecond, morph.Duration-morphFadeInDelay))
			if p > 1 {
				return 1
			}
			return p
		}
		if toAgent {
			if el < morphFadeOut {
				return 1 - float32(el)/float32(morphFadeOut)
			}
			if el < morphFadeInDelay+morphWsInDelay {
				return 0
			}
			p := float32(el-morphFadeInDelay-morphWsInDelay) / float32(max(time.Millisecond, morph.Duration-morphFadeInDelay-morphWsInDelay))
			if p > 1 {
				return 1
			}
			return p
		}
	case "nav", "chat":
		// 导航栏内容换成文件树 / 会话 = 左栏的 a-swap 入场(side.go + kit.SwapIn);
		// 对话栏的滑入 + 淡入(chat-dock / chat-center)作用在整块对话栏上(chatPane + chatSlide),
		// 内容这里不再另做一层淡入,免得叠两种过渡。
		return 1
	}
	return 1
}

// morphWrap 包住形态切换时要分轨淡入 / 淡出的内容区。
func (a *App) morphWrap(c *ui.Context, track string, inner func()) {
	col := ui.Column(c).Grow(1).MinHeight(0)
	if op := a.morphOpacity(track); op < 1 {
		col.Opacity(op)
		if a.morphing() {
			a.frameSoon()
		}
	}
	col.Children(inner)
}
