package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

var sessRestoreLog = diag.L("session.restore")

const (
	pRestore      = "general.restore"
	lastStateFile = "last-session.json"
	// 演示模式单独存一份,避免把真实项目的标签 / 会话恢复进演示(显示「无法打开」)。
	lastStateFileDemo = "last-session-demo.json"
)

// lastRuntime 关闭时落盘的运行状态(仅 general.restore 开启时读写)。
type lastRuntime struct {
	Session    string   `json:"session,omitempty"`
	Project    string   `json:"project,omitempty"`
	Workspace  string   `json:"workspace,omitempty"`
	NavAgent   string   `json:"navAgent,omitempty"`
	NavIDE     string   `json:"navIDE,omitempty"`
	EditorTabs []string `json:"editorTabs,omitempty"`
	PinnedTabs []string `json:"pinnedTabs,omitempty"`
	ActiveTab  string   `json:"activeTab,omitempty"`
	BottomTab  string   `json:"bottomTab,omitempty"`

	// 以下是「关闭时的窗口状态」:编辑器拆分、光标位置、Markdown 视图、树展开、项目展开、终端标签数、输入框草稿
	SplitDir  int8              `json:"splitDir,omitempty"`
	SplitFrac float32           `json:"splitFrac,omitempty"`
	Tabs2     []string          `json:"tabs2,omitempty"`
	Active2   string            `json:"active2,omitempty"`
	PaneSel   int8              `json:"paneSel,omitempty"`
	Cursors   map[string][2]int `json:"cursors,omitempty"` // 路径 → 行、列(0 起)
	MdMode    map[string]int8   `json:"mdMode,omitempty"`  // 路径 → Markdown 视图(手动选的)
	TreeOpen  []string          `json:"treeOpen,omitempty"`
	ProjOpen  []string          `json:"projOpen,omitempty"`
	TermCount int               `json:"termCount,omitempty"`
	TermSel   int               `json:"termSel,omitempty"`
	Draft     string            `json:"draft,omitempty"`
}

func lastStatePath(demo bool) string {
	if demo {
		return filepath.Join(diag.Home(), lastStateFileDemo)
	}
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
	st.Cursors, st.MdMode = map[string][2]int{}, map[string]int8{}
	collect := func(tabs []*ideTab) (paths []string) {
		for _, tb := range tabs {
			if tb == nil || tb.virt != "" || tb.diff {
				continue
			}
			p := strings.TrimSpace(tb.path)
			if p == "" {
				continue
			}
			paths = append(paths, p)
			if tb.st != nil && tb.buf != nil { // 没画过的标签没有编辑器状态,光标无从谈起
				pos := tb.st.CursorPos()
				st.Cursors[p] = [2]int{pos.Line, pos.Col}
			}
			if tb.mdMode != 0 {
				st.MdMode[p] = tb.mdMode
			}
		}
		return
	}
	st.EditorTabs = collect(a.ide.tabs)
	for _, tb := range a.ide.tabs {
		if tb != nil && tb.pinned && isFileTab(tb) {
			st.PinnedTabs = append(st.PinnedTabs, tb.path)
		}
	}
	for _, tb := range a.ide.tabs2 {
		if tb != nil && tb.pinned && isFileTab(tb) {
			st.PinnedTabs = append(st.PinnedTabs, tb.path)
		}
	}
	if t := a.ide.tab(a.ide.active); t != nil && t.virt == "" && !t.diff {
		st.ActiveTab = t.path
	}
	if a.ide.splitDir != 0 {
		st.SplitDir, st.SplitFrac, st.PaneSel = a.ide.splitDir, a.ide.splitFrac, a.ide.paneSel
		st.Tabs2 = collect(a.ide.tabs2)
		for _, tb := range a.ide.tabs2 {
			if tb != nil && tb.id == a.ide.active2 {
				st.Active2 = tb.path
			}
		}
	}
	for p, on := range a.ide.open {
		if on {
			st.TreeOpen = append(st.TreeOpen, p)
		}
	}
	for k, on := range a.projOpen {
		if on {
			st.ProjOpen = append(st.ProjOpen, k)
		}
	}
	started := 0
	for _, tp := range a.ide.tps {
		if tp != nil && tp.t != nil {
			started++
		}
	}
	st.TermCount, st.TermSel = started, a.ide.tpSel // 只记「真起过 shell」的标签数;进程本身不能恢复,下次显示时各起一个新的
	if !a.runDemo {
		st.Draft = a.draft
	}
	sessRestoreLog.Info("保存运行状态", "session", st.Session, "project", st.Project, "tabs", len(st.EditorTabs))
	end := diag.SpanSlow("session", "保存运行状态", 100*time.Millisecond)
	defer end()
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		sessRestoreLog.Warn("序列化运行状态失败", "err", err)
		return
	}
	path := lastStatePath(a.runDemo)
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
	b, err := os.ReadFile(lastStatePath(a.runDemo))
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
	pinned := map[string]bool{}
	for _, p := range st.PinnedTabs {
		pinned[p] = true
	}
	for _, p := range st.EditorTabs {
		a.ide.openFile(p, true)
		if pinned[p] {
			if t := a.ide.tabByPath(p); t != nil {
				t.pinned = true
			}
		}
	}
	a.ide.tabs = reorderPinnedTabs(a.ide.tabs)
	if st.ActiveTab != "" {
		if t := a.ide.tabByPath(st.ActiveTab); t != nil {
			a.ide.active = t.id
		}
	}
	a.applySavedWindowState(st)
	sessRestoreLog.Info("恢复运行状态", "session", st.Session, "tabs", len(st.EditorTabs))
	return strings.TrimSpace(st.Session)
}

// applySavedWindowState 恢复编辑器拆分、光标位置、Markdown 视图、树 / 项目展开、终端标签数与输入框草稿。
func (a *App) applySavedWindowState(st lastRuntime) {
	d := a.ide
	for p, m := range st.MdMode {
		if t := d.tabByPath(p); t != nil {
			t.mdMode = m
		}
	}
	for p, lc := range st.Cursors { // Markdown 标签带 reveal 会被强制切源码,所以只恢复非 Markdown 文件的光标
		if t := d.tabByPath(p); t != nil && t.buf != nil && !isMarkdownFile(p) {
			t.reveal = &[3]int{lc[0], lc[1], lc[1]}
		}
	}
	if st.SplitDir != 0 && len(st.Tabs2) > 0 {
		d.splitDir, d.splitFrac, d.paneSel = st.SplitDir, st.SplitFrac, 0
		for _, p := range st.Tabs2 {
			if t := d.tabByPath(p); t != nil && t.buf != nil {
				a.openSharedInPane(1, t) // 同一文件两组共享缓冲区
			} else {
				d.addTabTo(1, a.mkFileTab(p, false))
			}
		}
		for _, t := range d.tabs2 {
			if t.path == st.Active2 {
				d.active2 = t.id
			}
		}
		if d.active2 == "" && len(d.tabs2) > 0 {
			d.active2 = d.tabs2[0].id
		}
		d.paneSel = st.PaneSel
	}
	if len(st.TreeOpen) > 0 {
		if d.open == nil {
			d.open = map[string]bool{}
		}
		for _, p := range st.TreeOpen {
			d.open[p] = true
		}
	}
	for _, k := range st.ProjOpen {
		if a.projOpen == nil {
			a.projOpen = map[string]bool{}
		}
		a.projOpen[k] = true
	}
	if st.TermCount > 1 && !a.runDemo { // 终端标签数:壳是下次显示面板时才起的,这里只补出空标签位
		for len(d.tps) < st.TermCount {
			d.tps = append(d.tps, &termPanel{})
		}
		d.tpSel = min(max(st.TermSel, 0), len(d.tps)-1)
	}
	if st.Draft != "" && !a.runDemo && a.draft == "" {
		a.draft = st.Draft
	}
}
