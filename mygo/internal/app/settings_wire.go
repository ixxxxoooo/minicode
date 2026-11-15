// 设置页偏好 → 运行时接线(B6)。键名与 settings_tabs.go 对齐。
package app

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/claude"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/term"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

const (
	pSysFollow   = "appearance.system"
	pWaitAnim    = "appearance.waitAnim"
	pScrollbar   = "appearance.scrollbar"
	pPermMode    = "perm.mode"
	pPermNoByp   = "perm.noBypass"
	pPermDirs    = "perm.dirs"
	pRuntimeBin  = "runtime.bin"
	pChatSys     = "chat.sys"
	pPrivacyLvl  = "privacy.level"
	pAdvLiveOut  = "adv.liveOut"
	pAdvWatchdog = "adv.watchdog"
	pAdvFlowMeas = "adv.flowMeasure"
	pScmDiffMax  = "scm.diffMax"
	pScmFetch    = "scm.fetch"
	pScmFetchMin = "scm.fetchMin"
	pAboutAuto   = "about.autoUpdate"
	pAboutChan   = "about.channel"
)

// defaultModelIDs 与设置页「默认模型」胶囊顺序一致。
var defaultModelIDs = []string{"default", "opus", "sonnet", "haiku"}

var defaultModelLabels = []string{"Fable 5.2", "Opus 5.5", "Sonnet 5.5", "Haiku 5.5"}

// MigrateSettingsPrefs 把已废弃的键迁移到当前键(只执行一次,有旧键且新键空时拷贝)。
func (a *App) MigrateSettingsPrefs() {
	if a == nil || a.prefs == nil {
		return
	}
	if a.prefs.Has("models.model") && !a.prefs.Has(prefModel) {
		ix := a.prefs.Int("models.model", 2)
		if ix >= 0 && ix < len(defaultModelIDs) {
			a.prefs.Set(prefModel, defaultModelIDs[ix])
		}
	}
	if a.prefs.Has("models.effort") && !a.prefs.Has(prefEffort) {
		a.prefs.Set(prefEffort, a.prefs.Int("models.effort", 2))
	}
	if a.prefs.Has("chat.mode") && !a.prefs.Has(pPermMode) {
		a.prefs.Set(pPermMode, a.prefs.Int("chat.mode", 0))
	}
}

func permModeIndex(m core.PermissionMode) int {
	switch m {
	case core.ModeDefault:
		return 0
	case core.ModeAcceptEdits:
		return 1
	case core.ModePlan:
		return 2
	default:
		return 0
	}
}

func permModeFromIndex(i int) core.PermissionMode {
	switch i {
	case 1:
		return core.ModeAcceptEdits
	case 2:
		return core.ModePlan
	default:
		return core.ModeDefault
	}
}

// DefaultPermMode 是新会话的初始权限模式(设置 perm.mode)。
func (a *App) DefaultPermMode() core.PermissionMode {
	if a == nil || a.prefs == nil {
		return core.ModeDefault
	}
	return permModeFromIndex(a.prefs.Int(pPermMode, 0))
}

// NoBypassPermissions 为真时禁止切换到「跳过权限确认」。
func (a *App) NoBypassPermissions() bool {
	if a == nil || a.prefs == nil {
		return false
	}
	return a.prefs.Bool(pPermNoByp, false)
}

func splitDirList(raw string) []string {
	raw = strings.ReplaceAll(raw, "\n", ",")
	var out []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "~/") {
			if h, err := os.UserHomeDir(); err == nil {
				p = filepath.Join(h, p[2:])
			}
		}
		out = append(out, filepath.Clean(p))
	}
	return out
}

// ExtraPermDirs 是设置里声明的额外可访问目录(--add-dir)。
func (a *App) ExtraPermDirs() []string {
	if a == nil || a.prefs == nil {
		return nil
	}
	return splitDirList(a.prefs.Str(pPermDirs, ""))
}

// FillClaudeConfig 用偏好填充真实后端的启动配置(新进程启动时生效)。
func (a *App) FillClaudeConfig(cfg *claude.Config) {
	if a == nil || a.prefs == nil || cfg == nil {
		return
	}
	a.MigrateSettingsPrefs()
	cfg.ClaudePath = strings.TrimSpace(a.prefs.Str(pRuntimeBin, ""))
	cfg.AppendSystemPrompt = a.prefs.Str(pChatSys, "")
	if m := strings.TrimSpace(a.prefs.Str(prefModel, "")); m != "" {
		cfg.Model = m
	}
	if a.prefs.Has(prefEffort) {
		e := core.Effort(a.prefs.Int(prefEffort, int(core.EffortMedium)))
		if e >= core.EffortLow && e <= core.EffortMax {
			cfg.Effort = &e
		}
	}
	mode := a.DefaultPermMode()
	cfg.Mode = &mode
	seen := map[string]bool{}
	for _, d := range cfg.AddDirs {
		seen[d] = true
	}
	for _, d := range a.ExtraPermDirs() {
		if !seen[d] {
			cfg.AddDirs = append(cfg.AddDirs, d)
			seen[d] = true
		}
	}
}

// RuntimeBinValid 检查设置里的 claude 可执行文件是否存在(空 = 自动查找,视为有效)。
func (a *App) RuntimeBinValid() (bool, string) {
	if a == nil || a.prefs == nil {
		return true, ""
	}
	raw := strings.TrimSpace(a.prefs.Str(pRuntimeBin, ""))
	if raw == "" {
		_, ok := claude.FindClaude("")
		return ok, ""
	}
	p, ok := claude.FindClaude(raw)
	if !ok {
		return false, "找不到可执行文件"
	}
	return true, p
}

// DiffMaxPref 返回 Diff 大小上限字符串。
func (a *App) DiffMaxPref() string {
	if a == nil || a.prefs == nil {
		return "512 KB"
	}
	return a.prefs.Str(pScmDiffMax, "512 KB")
}

// LiveTaskOutput 是否开启前台命令实时尾读。
func (a *App) LiveTaskOutput() bool {
	if a == nil || a.prefs == nil {
		return true
	}
	return a.prefs.Bool(pAdvLiveOut, true)
}

// FlowMeasureAll 是否在载入长会话时一次测完行高。
func (a *App) FlowMeasureAll() bool {
	if a == nil || a.prefs == nil {
		return true
	}
	return a.prefs.Bool(pAdvFlowMeas, true)
}

// ApplyDiagPrefs 应用日志级别与看门狗开关(applyPrefs 末尾调用)。
func (a *App) ApplyDiagPrefs() {
	if a == nil || a.prefs == nil {
		return
	}
	if os.Getenv("MINI_LOG") == "" {
		lv := []diag.Level{diag.LevelError, diag.LevelWarn, diag.LevelInfo, diag.LevelDebug}
		i := a.prefs.Int(pPrivacyLvl, 2)
		if i < 0 || i >= len(lv) {
			i = 2
		}
		diag.SetLevel(lv[i])
	}
	diag.SetWatchdogEnabled(a.prefs.Bool(pAdvWatchdog, true))
}

// ApplySystemAppearance 跟随系统深浅色时按配对切换主题(不改变用户选的 appearance.theme 键值)。
func (a *App) ApplySystemAppearance() {
	if a == nil || a.prefs == nil || !a.prefs.Bool(pSysFollow, false) {
		return
	}
	baseID := a.prefs.Str(kTheme, a.pal.ID)
	sysDark := systemIsDark()
	pair := themePairFor(baseID, sysDark)
	if pair != "" && pair != a.pal.ID {
		a.pal = theme.Find(pair)
	}
}

// themePairFor 返回在系统深浅下应使用的主题 id。
func themePairFor(base string, sysDark bool) string {
	lightDark := map[string]string{
		"paper": "ink", "mist": "graphite", "celadon": "aurora",
		"ink": "paper", "graphite": "mist", "aurora": "celadon",
	}
	p := theme.Find(base)
	if p.Dark == sysDark {
		return base
	}
	if t, ok := lightDark[base]; ok {
		return t
	}
	if sysDark {
		return "ink"
	}
	return "paper"
}

// TermStartOpts 供 term_panel 调用:登录 shell、项目目录、shell 与回看(新终端生效)。
func (a *App) TermStartOpts(cols, rows, sb int, cwd string) term.Options {
	o := term.Options{Cols: cols, Rows: rows, Scrollback: sb, LoginShell: true, Cwd: cwd, Notify: a.redraw}
	if a == nil || a.prefs == nil {
		return o
	}
	o.LoginShell = a.prefs.Bool("term.login", true)
	if !a.prefs.Bool("term.cwd", true) {
		o.Cwd = ""
	} else if cwd != "" {
		o.Cwd = cwd
	}
	if sh := a.prefs.Str("term.shell", ""); sh != "" {
		o.Shell = sh
	}
	if raw := a.prefs.Str("term.scrollback", ""); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			o.Scrollback = v
		}
	}
	return o
}

func modelChipIndex(id string) int {
	for i, m := range defaultModelIDs {
		if m == id {
			return i
		}
	}
	return 2
}
