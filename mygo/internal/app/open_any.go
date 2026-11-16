package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// resolveUserPath 展开 ~/ 与相对路径(相对当前项目根),得到绝对路径。
func (a *App) resolveUserPath(p string) (string, bool) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", false
	}
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", false
		}
		switch p {
		case "~":
			p = home
		case "~/", "~\\":
			p = home
		default:
			p = filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(p, "~/"), "~\\"))
		}
	} else if !filepath.IsAbs(p) {
		if a.proj == "" {
			return "", false
		}
		p = filepath.Join(a.proj, p)
	}
	p = filepath.Clean(p)
	if strings.Contains(p, "..") {
		return "", false
	}
	return p, true
}

// pathDisplay 面包屑用:工作区外路径尽量用 ~ 缩写。
func (a *App) pathDisplay(abs string) string {
	abs = filepath.Clean(abs)
	if !workspace.IsExternalPath(abs) {
		return abs
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if abs == home {
			return "~"
		}
		if rest, ok := strings.CutPrefix(abs, home+string(filepath.Separator)); ok {
			return "~/" + rest
		}
	}
	return abs
}

// openAnyFile 在工作区内或磁盘上打开文件;行号从 1 起(0 表示不跳转)。
func (a *App) openAnyFile(p string, line, col int) bool {
	if a.ide == nil || a.ide.fs == nil {
		return false
	}
	line = max(line, 0)
	col = max(col, 0)
	if rel := a.wsRel(p); rel != "" {
		if _, err := a.ide.fs.Read(rel); err == nil {
			a.screen = ScreenWork
			a.jumpTo(rel, max(line-1, 0), max(col-1, 0), 0)
			a.revealWorkspace()
			log.Info("打开工作区文件", "path", rel)
			return true
		}
	}
	abs, ok := a.resolveUserPath(p)
	if !ok {
		return false
	}
	st, err := os.Stat(abs)
	if err != nil {
		a.toastOpenFileErr(abs, err)
		return false
	}
	if st.IsDir() {
		a.toast = "这是文件夹,请用「添加项目」打开:" + filepath.Base(abs)
		return false
	}
	if _, err := a.ide.fs.Read(abs); err != nil {
		a.toastOpenReadErr(abs, err)
		return false
	}
	a.screen = ScreenWork
	if line > 0 || col > 0 {
		a.openFileAtLine(abs, max(line-1, 0), max(col-1, 0))
	} else {
		a.ide.openFile(abs, true)
	}
	a.revealWorkspace()
	log.Info("打开文件", "path", abs, "external", workspace.IsExternalPath(abs))
	return true
}

func (a *App) toastOpenFileErr(abs string, err error) {
	base := filepath.Base(abs)
	switch {
	case errors.Is(err, os.ErrNotExist):
		a.toast = "找不到文件:" + base
	case errors.Is(err, os.ErrPermission):
		a.toast = "没有权限打开:" + base
	default:
		a.toast = "无法打开:" + base
	}
}

func (a *App) toastOpenReadErr(abs string, err error) {
	base := filepath.Base(abs)
	if errors.Is(err, workspace.ErrTooLarge) {
		a.toast = "文件过大,不能用编辑器打开"
		return
	}
	if errors.Is(err, os.ErrNotExist) {
		a.toast = "找不到文件:" + base
		return
	}
	a.toast = "无法打开:" + err.Error()
}

// paletteAbsFile 命令面板里输入绝对路径 / ~/… 时,在列表顶部提供打开项。
func (a *App) paletteAbsFile(q string) *scored {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil
	}
	pathQ, line, col, lined := parsePathLine(q)
	if lined && pathQ != "" {
		q = pathQ
	}
	if !strings.HasPrefix(q, "/") && !strings.HasPrefix(q, "~") {
		return nil
	}
	abs, ok := a.resolveUserPath(q)
	if !ok {
		return nil
	}
	st, err := os.Stat(abs)
	if err != nil || st.IsDir() {
		return nil
	}
	name := filepath.Base(abs)
	item := palItem{
		group: "文件",
		icon:  fileIcon(name),
		title: name,
		right: a.pathDisplay(abs),
		path:  abs,
		desc:  "打开磁盘上的文件",
		run: func() {
			if lined {
				a.openFileAtLine(abs, line, col)
			} else {
				a.paletteOpenFile(abs, a.pq.query)
			}
			a.SetMode(ModeIDE)
		},
	}
	return &scored{palItem: item, score: 3000, ord: -1}
}
