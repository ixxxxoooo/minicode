package app

import "strings"

const pQueueAtt = "chat.queueAtt"

// queueDraftFor 按 chat.queueAtt 决定排队/发出时的正文:关闭时不带图片语法与 @ 引用。
func (a *App) queueDraftFor(text string) string {
	if a == nil || a.prefs == nil || a.prefs.Bool(pQueueAtt, true) {
		return text
	}
	body, _ := splitDraftImages(text)
	return strings.TrimSpace(stripAtRefs(body))
}

func stripAtRefs(s string) string {
	var b strings.Builder
	for _, tok := range strings.Fields(s) {
		if strings.HasPrefix(tok, "@") {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(tok)
	}
	return b.String()
}

// qTurnEnded 回合结束后的队列策略;chat.queuePause 关闭时中断也继续自动发送。
func (a *App) qTurnEnded(normal bool) {
	if a == nil || a.prefs == nil || a.prefs.Bool("chat.queuePause", true) {
		a.q.turnEnded(normal)
		return
	}
	a.q.turnEnded(true)
}
