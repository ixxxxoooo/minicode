package app

import (
	"strings"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// afterEventCanvas 回合结束后尝试把助手回复里的画布块写回当前画布页。
func (a *App) afterEventCanvas(ev core.Event) {
	if _, ok := ev.(core.EvTurnEnded); !ok {
		return
	}
	if a.screen != ScreenCanvas {
		return
	}
	text := lastAssistantText(a.st.Items)
	if text == "" {
		return
	}
	if !strings.Contains(text, "canvas") && !strings.Contains(text, `"blocks"`) {
		return
	}
	s := a.cvStore()
	if s == nil {
		return
	}
	if ok, msg := s.ApplyClaudeReply(text, "Claude", "按对话修改", a.now()); ok {
		a.canvasPersist()
		a.cv.preview = false
		a.cv.verIx = 0
		a.cv.st.Y = 0
		a.toast = "已从对话写入画布 v" + itoa(s.Versions[0].N)
		log.Info("画布自动写回", "ver", s.Versions[0].N)
	} else if msg != "" && msg != "内容与当前版本相同" {
		log.Debug("画布自动写回跳过", "reason", msg)
	}
}

func lastAssistantText(items []core.Item) string {
	for i := len(items) - 1; i >= 0; i-- {
		if as, ok := items[i].(*core.Assistant); ok && strings.TrimSpace(as.Text) != "" && !as.Streaming {
			return as.Text
		}
	}
	return ""
}
