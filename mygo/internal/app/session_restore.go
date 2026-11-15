package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

var sessRestoreLog = diag.L("session.restore")

const (
	pRestore      = "general.restore"
	lastStateFile = "last-session.json"
)

// lastRuntime 关闭时落盘的运行状态(仅 general.restore 开启时读写)。
type lastRuntime struct {
	Session    string   `json:"session,omitempty"`
	Project    string   `json:"project,omitempty"`
	Workspace  string   `json:"workspace,omitempty"`
	NavAgent   string   `json:"navAgent,omitempty"`
	NavIDE     string   `json:"navIDE,omitempty"`
	EditorTabs []string `json:"editorTabs,omitempty"`
	ActiveTab  string   `json:"activeTab,omitempty"`
	BottomTab  string   `json:"bottomTab,omitempty"`
}

func lastStatePath() string {
	return filepath.Join(diag.Home(), lastStateFile)
}

// RestoreEnabled 是否恢复上次会话与布局。
func (a *App) RestoreEnabled() bool {
	if a == nil || a.prefs == nil || a.secondary {
		return false
	}
	return a.prefs.Bool(pRestore, true)
}

// SaveRuntimeState 把当前会话、项目、导航与编辑器标签写入 ~/.minicode/last-session.json。
func (a *App) SaveRuntimeState() {
	if !a.RestoreEnabled() {
		return
	}
	st := lastRuntime{
		Session:   a.curSess,
		Project:   a.proj,
		BottomTab: a.ide.bottomTab,
	}
	if a.ws != nil {
		st.Workspace = a.ws.ID
	}
	st.NavAgent, st.NavIDE = a.navBy[ModeAgent], a.navBy[ModeIDE]
	for _, tb := range a.ide.tabs {
		if tb == nil || tb.virt != "" || tb.diff {
			continue
		}
		if p := strings.TrimSpace(tb.path); p != "" {
			st.EditorTabs = append(st.EditorTabs, p)
		}
	}
	if t := a.ide.tab(a.ide.active); t != nil && t.virt == "" && !t.diff {
		st.ActiveTab = t.path
	}
	sessRestoreLog.Info("保存运行状态", "session", st.Session, "project", st.Project, "tabs", len(st.EditorTabs))
	end := diag.SpanSlow("session", "保存运行状态", 0, nil)
	defer end()
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		sessRestoreLog.Warn("序列化运行状态失败", "err", err)
		return
	}
	path := lastStatePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		sessRestoreLog.Warn("创建目录失败", "err", err)
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		sessRestoreLog.Warn("写入运行状态失败", "err", err)
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		sessRestoreLog.Warn("落盘运行状态失败", "err", err)
	}
}

// ApplySavedRuntime 启动时恢复导航与编辑器标签;返回应打开的会话 id(空 = 新会话 / 欢迎页)。
func (a *App) ApplySavedRuntime() string {
	if !a.RestoreEnabled() {
		return ""
	}
	if v := strings.TrimSpace(os.Getenv("MINI_SESSION")); v != "" {
		return v
	}
	b, err := os.ReadFile(lastStatePath())
	if err != nil {
		if !os.IsNotExist(err) {
			sessRestoreLog.Warn("读取运行状态失败", "err", err)
		}
		return ""
	}
	var st lastRuntime
	if err := json.Unmarshal(b, &st); err != nil {
		sessRestoreLog.Warn("运行状态损坏", "err", err)
		return ""
	}
	if st.NavAgent != "" {
		a.navBy[ModeAgent] = st.NavAgent
	}
	if st.NavIDE != "" {
		a.navBy[ModeIDE] = st.NavIDE
	}
	a.nav = a.navBy[a.mode]
	if st.BottomTab != "" {
		a.ide.bottomTab = st.BottomTab
	}
	if st.Workspace != "" && a.reg != nil {
		if w, ok := a.reg.Workspace(st.Workspace); ok {
			a.ActivateWorkspace(w.ID)
		}
	}
	if st.Project != "" && !a.runDemo {
		a.SetProject(st.Project)
	}
	for _, p := range st.EditorTabs {
		a.ide.openFile(p, true)
	}
	if st.ActiveTab != "" {
		if t := a.ide.tabByPath(st.ActiveTab); t != nil {
			a.ide.active = t.id
		}
	}
	sessRestoreLog.Info("恢复运行状态", "session", st.Session, "tabs", len(st.EditorTabs))
	return strings.TrimSpace(st.Session)
}
