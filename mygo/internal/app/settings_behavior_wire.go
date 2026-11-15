package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/ixxxxoooo/minicode/internal/claudecfg"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diag"
)

var (
	winCloseHooked bool
	autoCmpAt      time.Time
)

var behWireLog = diag.L("settings.wire")

// applySettingsRuntime 每帧/每次 applyBehavior 末尾:窗口、声音、自动压缩、队列、提示音。
func (a *App) applySettingsRuntime() {
	a.ensureWindowCloseHook()
	a.tickAutoCompact()
}

func (a *App) ensureWindowCloseHook() {
	if a == nil || a.win == nil || winCloseHooked || a.secondary {
		return
	}
	winCloseHooked = true
	a.win.OnClose(func(e *mygo.CloseEvent) {
		if a.prefs.Int(pWinClose, 0) == 0 {
			e.PreventDefault()
			a.win.Hide()
			behWireLog.Info("关闭窗口", "action", "hide")
		}
	})
}

func (a *App) tickAutoCompact() {
	if a == nil || a.prefs == nil || a.st == nil || !a.prefs.Bool(pModelCmp, true) {
		return
	}
	if a.st.Phase == core.PhaseRunning || a.ctxCompacting {
		return
	}
	r := a.st.Usage.ContextRatio()
	if r < a.ContextCompactThreshold() {
		return
	}
	now := time.Now()
	if !autoCmpAt.IsZero() && now.Sub(autoCmpAt) < 45*time.Second {
		return
	}
	autoCmpAt = now
	behWireLog.Info("自动压缩", "ratio", r, "thr", a.ContextCompactThreshold())
	a.requestCompact()
}

// SyncClaudeSettingsFromPrefs 把权限/思考偏好写入 Claude 设置文件(新会话前调用)。
func (a *App) SyncClaudeSettingsFromPrefs() {
	if a == nil || a.prefs == nil || a.demoWs() {
		return
	}
	layer := a.permLayer()
	// 项目 / 本地层的文件在用户仓库里:文件还不存在、用户也从没改过相关偏好时不创建,
	// 否则只是「打开一个项目的会话」就会在它的 .claude/ 里凭空多出文件(实测污染源代码管理的改动列表)。
	if !a.prefs.Has(pModelThink) && !a.prefs.Has(pPermMode) && !a.prefs.Has(pPermNoByp) && !a.prefs.Has(pPermDirs) &&
		!claudecfg.SettingsFileExists(layer, a.proj) {
		return
	}
	th := a.prefs.Bool(pModelThink, true)
	deny := a.prefs.Bool(pPermNoByp, false)
	sp := claudecfg.SessionPrefs{
		Layer:                 layer,
		Project:               a.proj,
		AlwaysThinking:        &th,
		DefaultMode:           claudecfg.PermModeCLI(a.prefs.Int(pPermMode, 0)),
		DisableBypass:         &deny,
		AdditionalDirectories: a.ExtraPermDirs(),
	}
	if err := claudecfg.SyncSessionPrefs(sp); err != nil {
		behWireLog.Warn("同步 Claude 设置失败", "err", err)
	}
}

// HandleComposerPaste 粘贴长文本时折叠为附件(general.paste);true=已折叠并写入 @ 引用(取消默认粘贴)。
func (a *App) HandleComposerPaste(text string) bool {
	if a == nil || a.prefs == nil || !a.prefs.Bool(pPasteFold, true) {
		return false
	}
	if len([]rune(text)) <= 2000 {
		return false
	}
	dir := filepath.Join(diag.Home(), "paste")
	_ = os.MkdirAll(dir, 0o700)
	name := fmt.Sprintf("%d.txt", time.Now().UnixNano())
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		behWireLog.Warn("写入粘贴附件失败", "err", err)
		return false
	}
	ref := "@" + path
	if a.draft != "" && !stringsHasSuffixNL(a.draft) {
		a.draft += "\n"
	}
	a.draft += ref + "\n"
	behWireLog.Info("粘贴折叠为附件", "chars", len([]rune(text)), "path", path)
	a.ux.focusDraft = true
	return true
}

func stringsHasSuffixNL(s string) bool {
	return strings.TrimSpace(s) == "" || s[len(s)-1] == '\n'
}

// expandPasteRefs 发送前把 ~/.minicode/paste 下的 @ 引用展开为 <pasted> 块。
func (a *App) expandPasteRefs(body string) string {
	if a == nil || body == "" {
		return body
	}
	pasteDir := filepath.Join(diag.Home(), "paste")
	var out []string
	for _, ln := range strings.Split(body, "\n") {
		raw := strings.TrimSpace(ln)
		if strings.HasPrefix(raw, "@") {
			p := strings.TrimPrefix(raw, "@")
			if rel, err := filepath.Rel(pasteDir, p); err == nil && !strings.HasPrefix(rel, "..") {
				if b, err := os.ReadFile(p); err == nil {
					out = append(out, "<pasted>\n"+string(b)+"\n</pasted>")
					continue
				}
			}
		}
		if raw != "" {
			out = append(out, ln)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
