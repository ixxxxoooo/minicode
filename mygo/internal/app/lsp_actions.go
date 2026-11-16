package app

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/lsp"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// 语言服务的「改动类」动作:重命名、格式化,以及转到类型定义 / 实现。

// lspApplyChanges 把各文件的编辑落到打开的缓冲区(可撤销)或磁盘;返回改了多少处、多少个文件。
func (a *App) lspApplyChanges(changes map[string][]lsp.TextEdit) (edits, files int, err error) {
	d := a.ide
	for uri, es := range changes {
		abs := lsp.URIToPath(uri)
		rel, rerr := filepath.Rel(a.proj, abs)
		if abs == "" || rerr != nil || strings.HasPrefix(rel, "..") || len(es) == 0 {
			continue
		}
		var src string
		tb := d.tab(rel)
		if tb != nil && tb.buf != nil {
			src = tb.buf.String()
		} else if s, e := d.fs.Read(rel); e == nil {
			src = s
		} else {
			err = e
			continue
		}
		out, e := lsp.ApplyEdits(src, es)
		if e != nil {
			err = e
			continue
		}
		if tb != nil && tb.buf != nil {
			tb.buf.Set(out)
		} else if e := d.fs.Write(rel, out); e != nil {
			err = e
			continue
		}
		edits += len(es)
		files++
	}
	d.invalidate()
	return
}

// lspRename 让服务器重命名光标处符号;成功返回 true(异步落地),没有服务器返回 false,服务器拒绝时调 fallback。
func (a *App) lspRename(tb *ideTab, off int, word, to string, fallback func()) bool {
	s := a.lspReadyFor(tb.path)
	if s == nil {
		return false
	}
	uri := lsp.PathToURI(a.absPath(tb.path))
	pos := lspPos(tb, off)
	a.lspJob("rename", s, func() func() {
		changes, err := s.c.Rename(uri, pos, to)
		return func() {
			if err != nil || len(changes) == 0 {
				if err != nil {
					a.outAppend(outLSP, "重命名被服务器拒绝:"+err.Error())
				}
				fallback()
				return
			}
			n, f, e := a.lspApplyChanges(changes)
			if e != nil {
				a.toast = "重命名部分失败:" + e.Error()
				return
			}
			a.toast = "已重命名 " + itoa(n) + " 处(" + itoa(f) + " 个文件):" + word + " → " + to
		}
	})
	return true
}

// formatDoc 格式化当前文档:服务器没有就提示。
func (a *App) formatDoc() {
	tb := a.ide.cur()
	if tb == nil || tb.buf == nil || tb.st == nil || tb.diskRO {
		return
	}
	s := a.lspReadyFor(tb.path)
	if s == nil {
		a.formatBuiltin(tb) // 没有可用的语言服务器:用内置格式化兜底
		return
	}
	uri := lsp.PathToURI(a.absPath(tb.path))
	tabSize, spaces := tb.st.TabSize, true
	if tabSize == 0 {
		tabSize = 4
	}
	before := tb.buf.Version()
	a.lspJob("format", s, func() func() {
		edits, err := s.c.Formatting(uri, tabSize, spaces)
		return func() {
			switch {
			case err != nil:
				a.toast = "格式化失败:" + err.Error()
			case len(edits) == 0:
				a.toast = "已经是格式化好的"
			case tb.buf.Version() != before:
				a.toast = "格式化期间文件被改动,已取消"
			default:
				out, e := lsp.ApplyEdits(tb.buf.String(), edits)
				if e != nil {
					a.toast = "格式化失败:" + e.Error()
					return
				}
				cur := tb.st.Cursor()
				tb.buf.Set(out)
				tb.st.Select(min(cur, len(out)), min(cur, len(out)))
				a.toast = "已格式化"
			}
		}
	})
}

// gotoKind 转到类型定义 / 实现(只有服务器能答)。
func (a *App) gotoKind(kind string) {
	tb := a.ide.cur()
	if tb == nil || tb.buf == nil || tb.st == nil {
		return
	}
	s := a.lspReadyFor(tb.path)
	if s == nil {
		a.toast = "需要语言服务器:" + kind + "只有服务器能回答"
		return
	}
	uri := lsp.PathToURI(a.absPath(tb.path))
	pos := lspPos(tb, tb.st.Cursor())
	a.lspJob("goto", s, func() func() {
		var locs []lsp.Location
		var err error
		if kind == "转到实现" {
			locs, err = s.c.Implementation(uri, pos)
		} else {
			locs, err = s.c.TypeDefinition(uri, pos)
		}
		return func() {
			if err != nil || len(locs) == 0 {
				a.toast = "没有找到" + strings.TrimPrefix(kind, "转到")
				return
			}
			rel, _ := a.locRel(locs[0])
			a.jumpToLSP(rel, locs[0].Range)
		}
	})
}

// lspEditorWhy 解释语言服务器菜单项为何不可用("" = 可用)。
func (a *App) lspEditorWhy(tb *ideTab) string {
	if tb == nil || tb.buf == nil {
		return "没有打开的文件"
	}
	if tb.diff || tb.virt != "" {
		return "差异 / 虚拟标签没有语言服务器"
	}
	if !a.lspOn() {
		return "设置里已关闭语言服务器"
	}
	if a.proj == "" {
		return "演示模式不启动真实语言服务器"
	}
	s := a.lspFor(tb.path)
	if s == nil {
		return "没有适用于此文件的语言服务器"
	}
	switch s.state {
	case lspReady:
		return ""
	case lspStarting:
		return "语言服务器正在启动…"
	case lspFailed:
		if s.err != "" {
			return s.err
		}
		return "语言服务器启动失败"
	default:
		return "语言服务器未就绪"
	}
}

// editorMenu 是编辑器右键菜单里追加的项(复制 / 粘贴 / 全选编辑器自带),分组与顺序对标 VS Code。
func (a *App) editorMenu(tb *ideTab, m *ui.Menu) {
	why := a.lspEditorWhy(tb)
	lspOff := why != ""
	m.Separator()
	if menuItem(m, "转到定义  F12").Chosen() {
		a.goDef()
	}
	it := menuItem(m, "转到类型定义")
	if lspOff {
		it.Disabled(true).Tooltip(why)
	} else if it.Chosen() {
		a.gotoKind("转到类型定义")
	}
	it = menuItem(m, "转到实现")
	if lspOff {
		it.Disabled(true).Tooltip(why)
	} else if it.Chosen() {
		a.gotoKind("转到实现")
	}
	if menuItem(m, "查找所有引用  ⇧F12").Chosen() {
		a.showRefs(a.cursorWord())
	}
	if menuItem(m, "预览定义  ⌥F12").Chosen() {
		a.openPeek()
	}
	m.Separator()
	it = menuItem(m, "重命名符号…  F2")
	if lspOff {
		it.Disabled(true).Tooltip(why)
	} else if it.Chosen() {
		a.openRename()
	}
	if menuItem(m, "格式化文档  ⇧⌥F").Chosen() {
		a.formatDoc()
	}
	selFmt := menuItem(m, "格式化选定内容")
	if lspOff {
		selFmt.Disabled(true).Tooltip(why)
	} else if selFmt.Chosen() {
		a.formatSelection()
	}
	it = menuItem(m, "源代码操作…  ⌘.")
	if lspOff {
		it.Disabled(true).Tooltip(why)
	} else if it.Chosen() {
		a.openActs()
	}
	m.Separator()
	if !tb.diff && tb.path != "" {
		if menuItem(m, "与 HEAD 比较").Chosen() {
			a.openDiffKind(dkHead, tb.path, true)
		}
	}
	on := a.prefs.Bool(pBlameInline, true)
	if menuItem(m, map[bool]string{true: "关闭行内 Git 追溯", false: "开启行内 Git 追溯"}[on]).Chosen() {
		if on {
			a.prefs.Set(pBlameInline, false)
		} else {
			a.prefs.Reset(pBlameInline)
		}
	}
	if menuItem(m, "显示文件历史").Chosen() {
		a.ide.sections["时间线"] = true
		a.nav, a.sideOpen = "files", true
	}
	m.Separator()
	if menuItem(m, "复制路径").Chosen() {
		a.copyText = a.absPath(tb.path)
		a.toast = "已复制路径"
	}
	if menuItem(m, "复制相对路径").Chosen() {
		a.copyText = tb.path
		a.toast = "已复制相对路径"
	}
	if menuItem(m, "在访达中显示").Disabled(a.proj == "").Tooltip(map[bool]string{true: "演示模式没有真实目录"}[a.proj == ""]).Chosen() {
		a.revealInFinder(a.absPath(tb.path))
	}
	m.Separator()
	wrap := a.prefs.Bool(pEdWrap, false)
	if menuItem(m, map[bool]string{true: "关闭自动换行", false: "开启自动换行"}[wrap]).Chosen() {
		a.prefs.Set(pEdWrap, !wrap)
	}
	if menuItem(m, "命令面板…  ⌘K").Chosen() {
		a.openPalette()
	}
	m.Separator()
	if menuItem(m, "问 Claude  ⌘L").Chosen() {
		a.askClaude("问")
	}
	if menuItem(m, "行内编辑…  ⌘I").Chosen() {
		a.openInline()
	}
	if menuItem(m, "让 Claude 解释").Chosen() {
		a.askClaude("解释")
	}
	if menuItem(m, "让 Claude 修改这段代码…").Chosen() {
		a.askClaude("编辑")
	}
	a.mcEditMenu(tb, m)
}

// absPath 项目相对路径 → 绝对路径(演示模式没有真实目录,原样返回)。
func (a *App) absPath(rel string) string {
	if workspace.IsExternalPath(rel) {
		if a.ide != nil && a.ide.fs != nil {
			return a.ide.fs.Abs(rel)
		}
		return filepath.Clean(rel)
	}
	if a.proj == "" {
		return rel
	}
	if a.multiRoot() {
		return a.ide.fs.Abs(rel)
	}
	return filepath.Join(a.proj, rel)
}

// revealInFinder 在访达里显示文件。
func (a *App) revealInFinder(abs string) {
	if testing.Testing() { // 单测里不要真的弹出访达
		return
	}
	if err := exec.Command("open", "-R", abs).Start(); err != nil {
		a.toast = "无法在访达中显示:" + err.Error()
	}
}
