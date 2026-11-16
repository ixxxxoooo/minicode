package app

import (
	"errors"
	"os"
	"path"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// 文件树里的新建 / 重命名 / 删除(设计稿 drag-more「文件操作」一节):
// 新建与重命名用树里的行内输入框,删除走确认框并移到废纸篓。

// fileOp 是文件树行内输入的状态。
type fileOp struct {
	kind   string // "new" | "newdir" | "rename" | ""(没有)
	dir    string // 新建:放在哪个文件夹里(相对项目根,"" = 根)
	path   string // 重命名:被改名的路径
	buf    string
	ext    string // 重命名时保留的扩展名
	err    string // 失败原因(不关闭输入行,显示在行下面)
	frames int    // 已渲染帧数:拿到焦点之前不判「点别处取消」
}

func (o fileOp) active() bool { return o.kind != "" }

// mutable 当前文件系统能不能改;不能返回原因。
func (a *App) mutable() (workspace.Mutable, string) {
	if m, ok := workspace.Unwrap(a.ide.fs).(workspace.Mutable); ok { // 先拆掉 WithExternal 包装,否则会被判成只读
		return m, ""
	}
	return nil, "这个项目是只读的"
}

func parentDir(p string) string {
	d := path.Dir(p)
	if d == "." || d == "/" {
		return ""
	}
	return d
}

// beginNew 在文件夹 dir 里开始新建(isDir = 新建文件夹),并展开它。
func (a *App) beginNew(dir string, isDir bool) {
	if a.multiRoot() && dir == "" { // 多根:顶层不是真实目录,建在主文件夹里
		dir = a.repos[0].name
	}
	if _, why := a.mutable(); why != "" {
		a.toast = why
		return
	}
	a.nav, a.sideOpen = "files", true
	for p := dir; p != ""; p = parentDir(p) {
		a.ide.open[p] = true
	}
	k := "new"
	if isDir {
		k = "newdir"
	}
	a.fo = fileOp{kind: k, dir: dir}
}

// beginRenameFile 开始重命名一个文件 / 文件夹。
func (a *App) beginRenameFile(p string) {
	if _, why := a.mutable(); why != "" {
		a.toast = why
		return
	}
	a.nav, a.sideOpen = "files", true
	base := path.Base(p)
	ext, name := "", base
	if e := path.Ext(base); e != "" {
		ext, name = e, strings.TrimSuffix(base, e)
	}
	a.fo = fileOp{kind: "rename", path: p, buf: name, ext: ext}
}

// opErrText 把文件系统的错误翻成界面上的一句话。
func opErrText(err error, name string) string {
	switch {
	case errors.Is(err, workspace.ErrExist):
		return "已存在同名文件「" + name + "」"
	case errors.Is(err, os.ErrPermission):
		return "没有权限"
	case errors.Is(err, os.ErrNotExist):
		return "找不到原文件(可能已被删除)"
	}
	return strings.TrimPrefix(err.Error(), workspace.ErrName.Error()+":")
}

// commitFileOp 提交行内输入:成功关闭输入行,失败留着并显示原因。
func (a *App) commitFileOp() {
	o := a.fo
	mu, why := a.mutable()
	if why != "" {
		a.fo.err = why
		return
	}
	name := strings.TrimSpace(o.buf)
	d := a.ide
	switch o.kind {
	case "new", "newdir":
		if err := workspace.ValidPath(name); err != nil {
			a.fo.err = opErrText(err, name)
			return
		}
		target := path.Join(o.dir, strings.TrimSuffix(name, "/"))
		if err := mu.Create(target, o.kind == "newdir"); err != nil {
			a.fo.err = opErrText(err, name)
			uxLog.Warn("新建失败", "path", target, "err", err)
			return
		}
		uxLog.Info("新建", "path", target, "dir", o.kind == "newdir")
		a.fo = fileOp{}
		a.afterFileChange()
		for p := parentDir(target); p != ""; p = parentDir(p) {
			d.open[p] = true
		}
		if o.kind == "new" {
			d.openFile(target, true)
			d.sel = target
		} else {
			d.open[target] = true
		}
		a.toast = "已新建 " + target
	case "rename":
		if err := workspace.ValidName(name); err != nil {
			a.fo.err = opErrText(err, name)
			return
		}
		fullName := name
		if o.ext != "" && !strings.HasSuffix(name, o.ext) {
			fullName = name + o.ext
		}
		if fullName == path.Base(o.path) {
			a.fo = fileOp{}
			return
		}
		to := path.Join(parentDir(o.path), fullName)
		if err := mu.Rename(o.path, to); err != nil {
			a.fo.err = opErrText(err, name)
			uxLog.Warn("重命名失败", "from", o.path, "to", to, "err", err)
			return
		}
		uxLog.Info("重命名", "from", o.path, "to", to)
		a.fo = fileOp{}
		a.renameTabs(o.path, to)
		a.afterFileChange()
		a.toast = "已重命名为 " + name
	}
}

// moveFile 把文件 / 文件夹移到另一个文件夹下(拖放)。
func (a *App) moveFile(from, dir string) {
	mu, why := a.mutable()
	if why != "" {
		a.toast = why
		return
	}
	if from == dir || strings.HasPrefix(dir+"/", from+"/") {
		a.toast = "不能移到自己的子文件夹里"
		return
	}
	to := path.Join(dir, path.Base(from))
	if err := mu.Rename(from, to); err != nil {
		a.toast = opErrText(err, path.Base(to))
		return
	}
	a.renameTabs(from, to)
	a.afterFileChange()
	a.toast = "已移动到 " + dir
}

// afterFileChange 磁盘上的文件结构变了:清树缓存、符号索引,刷新 Git 状态。
func (a *App) afterFileChange() {
	a.ide.listCache, a.ide.filesCache = nil, nil
	a.ide.invalidate()
	a.scmRefresh()
}

// renameTabs 文件 / 文件夹改名后,已打开的标签跟着改路径(保留未保存的修改);差异标签关掉。
func (a *App) renameTabs(from, to string) {
	d := a.ide
	remap := func(p string) (string, bool) {
		if p == from {
			return to, true
		}
		if strings.HasPrefix(p, from+"/") {
			return to + strings.TrimPrefix(p, from), true
		}
		return p, false
	}
	var drop []string
	for _, tb := range append(d.tabs, d.tabs2...) {
		if tb.virt != "" {
			continue
		}
		np, ok := remap(tb.path)
		if !ok {
			continue
		}
		if tb.diff {
			drop = append(drop, tb.id)
			continue
		}
		switch tb.id {
		case d.active:
			d.active = np
		case d.active2:
			d.active2 = np
		}
		tb.id, tb.path, tb.hooked = np, np, false
	}
	for _, id := range drop {
		a.closeWsTab(id)
	}
	if np, ok := remap(d.sel); ok {
		d.sel = np
	}
	for i, p := range d.closed {
		if np, ok := remap(p); ok {
			d.closed[i] = np
		}
	}
	if d.open[from] {
		delete(d.open, from)
		d.open[to] = true
	}
}

// askDelete 弹确认框,确认后把文件 / 文件夹移到废纸篓。
func (a *App) askDelete(p string, dir bool) {
	mu, why := a.mutable()
	if why != "" {
		a.toast = why
		return
	}
	name := path.Base(p)
	body := "可以在访达的废纸篓里放回原处。"
	if a.proj == "" {
		body = "演示数据:直接从内存项目里移除,没有废纸篓。"
	}
	title := "把 " + name + " 移到废纸篓?"
	if dir {
		if n := mu.Count(p); n > 0 {
			title = "把文件夹 " + name + "(连同里面的 " + itoa(n) + " 个文件)移到废纸篓?"
		}
	}
	dirty := 0
	for _, tb := range a.ide.tabs {
		if tb.virt == "" && !tb.diff && (tb.path == p || strings.HasPrefix(tb.path, p+"/")) && tb.dirty() {
			dirty++
		}
	}
	if dirty > 0 {
		body += "\n其中有 " + itoa(dirty) + " 个标签含未保存的修改,删除后这些修改会丢失。"
	} else {
		body += "已打开的标签会关闭。"
	}
	a.confirmDlg(title, body, map[bool]string{true: "移除", false: "移到废纸篓"}[a.proj == ""], true, func() { a.deleteFile(p) })
}

func (a *App) deleteFile(p string) {
	mu, why := a.mutable()
	if why != "" {
		a.toast = why
		return
	}
	a.bg("delete-file", func() func() {
		err := mu.Remove(p)
		return func() {
			if err != nil {
				a.toast = "删除失败:" + opErrText(err, path.Base(p))
				uxLog.Warn("删除失败", "path", p, "err", err)
				return
			}
			uxLog.Info("移到废纸篓", "path", p)
			var ids []string
			for _, tb := range a.ide.tabs {
				if tb.virt == "" && (tb.path == p || strings.HasPrefix(tb.path, p+"/")) {
					ids = append(ids, tb.id)
				}
			}
			for _, id := range ids {
				a.closeWsTab(id)
			}
			// 刚关掉的标签不算「最近关闭」:文件已经不在了
			a.ide.closed = filterOut(a.ide.closed, p)
			if a.ide.sel == p || strings.HasPrefix(a.ide.sel, p+"/") {
				a.ide.sel = ""
			}
			delete(a.ide.open, p)
			a.afterFileChange()
			a.toast = "已移到废纸篓:" + path.Base(p)
			if a.proj == "" {
				a.toast = "已移除(演示数据):" + path.Base(p)
			}
		}
	})
}

func filterOut(list []string, p string) []string {
	out := list[:0]
	for _, x := range list {
		if x != p && !strings.HasPrefix(x, p+"/") {
			out = append(out, x)
		}
	}
	return out
}

// fileOpInput 是树里的行内输入框(新建 / 重命名共用):⏎ 提交,Esc 或点别处取消。
// 重命名时只选中不含扩展名的部分由输入框默认全选代替(MyGo 输入框没有部分选中接口)。
func (a *App) fileOpInput(c *ui.Context, depth int, isDir bool, icon string) {
	t := a.tk
	a.fo.frames++
	key := "fo-" + a.fo.kind + "-" + a.fo.dir + a.fo.path
	ui.Column(c).Key(key + "-col").Shrink(0).Children(func() {
		ui.Row(c).Key(key + "-row").Height(24).PaddingStart(6 + float32(depth)*12).PaddingEnd(6).Gap(5).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(12, 12).Shrink(0)
			if isDir {
				kit.Ic(c, "folder", kit.S12)
			} else {
				kit.Ic(c, icon, kit.S12)
			}
			box := ui.Row(c).Key(key+"-box").Grow(1).MinWidth(0).Height(20).PaddingX(6).AlignItems(ui.Center).
				Background(t.Fill1).Radius(t.RSm()).Border(1, t.Ac)
			box.Children(func() {
				in := ui.TextInputBase(c.Key(key), &a.fo.buf).Grow(1).MinWidth(0).FontSize(theme.FsBody).
					Background(ui.Transparent).BorderWidth(0).Padding(0).FocusRing(false).AutoFocus()
				in.OnSubmit(a.commitFileOp)
				in.OnShortcut(0, ui.KeyEscape, func() { a.fo = fileOp{} })
			})
			if a.fo.frames > 3 && !box.FocusWithin() { // 点了别处:取消
				a.fo = fileOp{}
			}
		})
		if a.fo.err != "" {
			ui.Text(c, a.fo.err).FontSize(theme.FsCap).TextColor(t.Err).PaddingStart(6+float32(depth)*12+29).Margin(1, 0, 3, 0)
		}
	})
}

// newTargetDir 顶部「新建文件」按钮放在哪:选中文件夹里;选中文件则在它旁边;都没有就是根。
func (a *App) newTargetDir() string {
	d := a.ide
	if d.sel == "" {
		return ""
	}
	if _, ok := d.open[d.sel]; ok || d.open[d.sel] {
		return d.sel
	}
	for _, e := range d.listing(parentDir(d.sel)) {
		if e.Path == d.sel && e.Dir {
			return d.sel
		}
	}
	return parentDir(d.sel)
}
