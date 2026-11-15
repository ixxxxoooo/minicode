package app

import (
	"strings"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// lastUserID 最近一条用户消息的 id。
func (a *App) lastUserID() string {
	for i := len(a.st.Items) - 1; i >= 0; i-- {
		if u, ok := a.st.Items[i].(*core.User); ok {
			return u.ID
		}
	}
	return ""
}

func (a *App) reviewFooterText(files []chFile) string {
	viewed := 0
	for _, f := range files {
		if a.rv.viewed[f.path] || a.fileDecided(f) {
			viewed++
		}
	}
	parts := []string{"已查看 " + itoa(viewed) + " / " + itoa(len(files))}
	uid := a.lastUserID()
	if stats, ok := a.st.TurnStats[uid]; ok && !stats.Empty() {
		if lbl := stats.ModelLabel(); lbl != "" {
			parts = append(parts, lbl)
		}
		parts = append(parts, strings.Split(stats.ShortText(), " · ")...)
	}
	if d, ok := a.st.Durations[uid]; ok && d > 0 {
		parts = append(parts, core.FormatElapsed(int64(d)*1000))
	}
	parts = append(parts, a.now().Format("1月2日 15:04"))
	return strings.Join(parts, " · ")
}

func (a *App) reviewRewind() {
	uid := a.lastUserID()
	if uid == "" {
		a.toast = "没有可回退的回合"
		return
	}
	a.send(core.CmdRewindFiles{UUID: uid})
	log.Info("回退到回合之前", "user", uid)
	a.toast = "正在回退 Claude 改过的文件…"
}
