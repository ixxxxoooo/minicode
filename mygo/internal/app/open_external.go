package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/ixxxxoooo/minicode/internal/canvas"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/projects"
)

// 系统要求用本应用打开文件(访达双击 / 拖到 Dock 图标 / `open -a` / 命令行参数)。
// 设计稿「图标与品牌 · 文件类型」:三种自有类型各有去处,其余文件在当前项目内才能打开。

var extLog = diag.L("app.open")

// maxWorkspaceFile 工作区文件大小上限(正常只有几百字节)。
const maxWorkspaceFile = 1 << 20

// HoldExternalOpen 窗口出来之前先把要打开的文件排队(冷启动时双击文件,系统在窗口创建前就会通知);
// hold=false 时依次处理排队的文件。
func (a *App) HoldExternalOpen(hold bool) {
	a.extHold = hold
	if hold {
		return
	}
	q := a.extQueue
	a.extQueue = nil
	for _, p := range q {
		a.OpenExternalFile(p)
	}
}

// OpenExternalFile 按扩展名分发:.mcworkspace / .mccanvas / .mcsettings 走各自的导入,
// 其余文件落在当前项目里就在编辑器打开,否则提示。
func (a *App) OpenExternalFile(path string) {
	if a.secondary { // 从属窗口不接管系统打开请求
		return
	}
	if a.extHold {
		a.extQueue = append(a.extQueue, path)
		extLog.Info("应用未就绪,文件先排队", "path", path)
		return
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	st, err := os.Stat(abs)
	if err != nil {
		a.toast = "找不到文件:" + filepath.Base(abs)
		extLog.Warn("要打开的文件不存在", "path", abs, "err", err)
		return
	}
	ext := strings.ToLower(filepath.Ext(abs))
	extLog.Info("打开文件", "path", abs, "ext", ext, "dir", st.IsDir())
	switch {
	case st.IsDir():
		a.toast = "这是文件夹,请用「添加项目」打开:" + filepath.Base(abs)
	case ext == projects.WorkspaceFileExt:
		a.importWorkspaceFile(abs)
	case ext == ".mccanvas":
		a.importCanvasFile(abs)
	case ext == ".mcsettings":
		a.importSettingsFile(abs)
	default:
		if !a.openPathRef(abs) {
			a.toast = "「" + filepath.Base(abs) + "」不在当前项目里,请先打开它所在的项目"
		}
	}
	a.focusMain()
}

// importWorkspaceFile 读 .mcworkspace:跳过不存在的文件夹,同一组文件夹已有工作区就直接切过去,
// 否则新建并进入。
func (a *App) importWorkspaceFile(path string) {
	if a.demoWs() {
		a.toast = "演示模式不导入工作区(没有真实目录)"
		return
	}
	a.bg("open.workspace", func() func() {
		defer diag.SpanSlow("app", "读取工作区文件", 300_000_000)()
		w, err := readWorkspaceFile(path)
		return func() { a.applyImportedWorkspace(path, w, err) }
	})
}

func readWorkspaceFile(path string) (projects.Workspace, error) {
	st, err := os.Stat(path)
	if err != nil {
		return projects.Workspace{}, err
	}
	if st.Size() > maxWorkspaceFile {
		return projects.Workspace{}, errTooBig
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return projects.Workspace{}, err
	}
	return projects.ParseWorkspaceFile(b)
}

type bigErr struct{}

func (bigErr) Error() string { return "文件太大,不像是这种文件" }

var errTooBig = bigErr{}

func (a *App) applyImportedWorkspace(path string, w projects.Workspace, err error) {
	if err != nil {
		a.toast = "无法打开工作区文件:" + err.Error()
		extLog.Warn("工作区文件无效", "path", path, "err", err)
		return
	}
	var keep []projects.Folder
	for _, f := range w.Folders {
		if st, e := os.Stat(f.Path); e == nil && st.IsDir() {
			keep = append(keep, f)
		}
	}
	skipped := len(w.Folders) - len(keep)
	if len(keep) == 0 {
		a.toast = "工作区文件里的文件夹在这台电脑上都不存在"
		extLog.Warn("工作区文件夹全部缺失", "path", path, "folders", len(w.Folders))
		return
	}
	w.Folders = keep
	// 同一组文件夹(同顺序)已经有工作区:不重复建
	for _, ex := range a.registry().Workspaces() {
		if sameFolders(ex.Folders, keep) {
			a.screen = ScreenWork
			a.ActivateWorkspace(ex.ID)
			a.toast = "已有这个工作区,已切换到「" + ex.Name + "」"
			return
		}
	}
	saved, err := a.registry().SaveWorkspace(w)
	if err != nil {
		a.toast = "导入工作区失败:" + friendlyGitErr(err.Error())
		extLog.Warn("保存导入的工作区失败", "err", err)
		return
	}
	a.screen = ScreenWork
	a.ActivateWorkspace(saved.ID)
	msg := "已导入工作区「" + saved.Name + "」"
	if skipped > 0 {
		msg += "(跳过 " + itoa(skipped) + " 个不存在的文件夹)"
	}
	a.toast = msg
	extLog.Info("导入工作区", "id", saved.ID, "name", saved.Name, "folders", len(saved.Folders), "skipped", skipped)
}

func sameFolders(a, b []projects.Folder) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if projects.Clean(a[i].Path) != projects.Clean(b[i].Path) {
			return false
		}
	}
	return true
}

// importCanvasFile 把 .mccanvas 导入成一张新画布并打开画布页。
func (a *App) importCanvasFile(path string) {
	a.bg("open.canvas", func() func() {
		defer diag.SpanSlow("app", "导入画布文件", 300_000_000)()
		return func() { a.applyImportedCanvas(path) }
	})
}

func (a *App) applyImportedCanvas(path string) {
	l := a.cvLib()
	id, err := l.Import(path, a.now())
	if err != nil {
		a.toast = "无法导入画布:" + err.Error()
		extLog.Warn("导入画布失败", "path", path, "err", err)
		return
	}
	if s, e := l.Store(id); e == nil {
		a.canv = s
	}
	a.cv.preview = false
	a.screen = ScreenCanvas
	a.toast = "已导入画布「" + a.canv.Doc.Title + "」"
	extLog.Info("导入画布", "id", id, "path", path)
}

// exportCanvasFile 把一张画布导出成 .mccanvas(写到「下载」并在访达里显示)。
func (a *App) exportCanvasFile(l *canvas.Library, id, title string) {
	dir := downloadsDir()
	dest := uniqueFile(dir, safeFileName(title, "画布"), ".mccanvas")
	if err := l.Export(id, dest); err != nil {
		a.toast = "导出失败:" + err.Error()
		extLog.Warn("导出画布失败", "id", id, "err", err)
		return
	}
	extLog.Info("导出画布", "id", id, "dest", dest)
	a.toast = "已导出到「下载」:" + filepath.Base(dest)
	a.revealInFinder(dest)
}

// exportWorkspaceFile 把工作区导出成 .mcworkspace(写到「下载」并在访达里显示)。
func (a *App) exportWorkspaceFile(w projects.Workspace) {
	b, err := projects.MarshalWorkspaceFile(w)
	if err != nil {
		a.toast = "导出失败:" + err.Error()
		return
	}
	dest := uniqueFile(downloadsDir(), safeFileName(w.Name, "工作区"), projects.WorkspaceFileExt)
	if err := writeFileAtomic(dest, b); err != nil {
		a.toast = "导出失败:" + err.Error()
		extLog.Warn("导出工作区失败", "id", w.ID, "err", err)
		return
	}
	extLog.Info("导出工作区", "id", w.ID, "dest", dest)
	a.toast = "已导出到「下载」:" + filepath.Base(dest)
	a.revealInFinder(dest)
}

// downloadsDir 是导出文件的落点:~/Downloads(没有就用主目录)。
func downloadsDir() string {
	home, _ := os.UserHomeDir()
	d := filepath.Join(home, "Downloads")
	if st, err := os.Stat(d); err == nil && st.IsDir() {
		return d
	}
	return home
}

// safeFileName 把标题变成能放进文件名的字符串(去掉路径分隔符与控制字符),空则用兜底名。
func safeFileName(s, fallback string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case r < 32, r == '/', r == '\\', r == ':', r == '*', r == '?', r == '"', r == '<', r == '>', r == '|':
			return '-'
		}
		return r
	}, strings.TrimSpace(s))
	s = strings.Trim(s, ". -")
	if s == "" {
		return fallback
	}
	if r := []rune(s); len(r) > 60 {
		s = string(r[:60])
	}
	return s
}

// uniqueFile 在目录里找一个不存在的文件名:名字.ext、名字 2.ext、名字 3.ext…
func uniqueFile(dir, name, ext string) string {
	p := filepath.Join(dir, name+ext)
	for i := 2; ; i++ {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return p
		}
		p = filepath.Join(dir, name+" "+itoa(i)+ext)
	}
}

// writeFileAtomic 先写临时文件再改名,中途失败不留半截文件。
func writeFileAtomic(dest string, b []byte) error {
	tmp := dest + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, dest)
}

// focusMain 把应用提到前台(双击文件后用户应当看到结果);单测 / 没有窗口时什么都不做。
func (a *App) focusMain() {
	if testing.Testing() || a.win == nil {
		return
	}
	mygo.App.Focus()
}
