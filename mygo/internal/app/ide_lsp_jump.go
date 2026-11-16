package app

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/lsp"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

var extOpenLog = diag.L("ide.extopen")

// 超过此大小打开前提示;超过 workspace.MaxRead 则拒绝。
const externalOpenWarn = 2 << 20

// jumpToLSP 跳到 LSP 范围(列换成字节列需要文本;先读文本)。
func (a *App) jumpToLSP(rel string, r lsp.Range) {
	if workspace.IsExternalPath(rel) {
		a.jumpToLSPAbs(rel, r)
		return
	}
	text, _ := a.ide.fileText(rel)
	lines := strings.Split(text, "\n")
	col := func(line, u16 int) int {
		if line >= 0 && line < len(lines) {
			return lsp.ColFromUTF16(lines[line], u16)
		}
		return 0
	}
	c0 := col(r.Start.Line, r.Start.Character)
	c1 := c0
	if r.End.Line == r.Start.Line {
		c1 = col(r.End.Line, r.End.Character)
	}
	a.jumpTo(rel, r.Start.Line, c0, c1)
}

func (a *App) jumpToLSPAbs(abs string, r lsp.Range) {
	abs = filepath.Clean(abs)
	if a.ide == nil {
		return
	}
	// 已打开:直接定位,不再读盘。
	if t := a.ide.tab(abs); t != nil && t.buf != nil {
		c0, c1 := a.lspColsAt(abs, r, t.buf.String())
		a.ide.navMark()
		a.ide.openAt(abs, r.Start.Line, c0, c1, true)
		a.ide.sel = abs
		a.ide.navPush(navPos{abs, r.Start.Line, c0})
		a.SetMode(ModeIDE)
		a.revealWorkspace()
		return
	}
	a.tryOpenDiskReadOnly(abs, r, false)
}

func (a *App) lspColsAt(abs string, r lsp.Range, text string) (c0, c1 int) {
	lines := strings.Split(text, "\n")
	col := func(line, u16 int) int {
		if line >= 0 && line < len(lines) {
			return lsp.ColFromUTF16(lines[line], u16)
		}
		return 0
	}
	c0 = col(r.Start.Line, r.Start.Character)
	c1 = c0
	if r.End.Line == r.Start.Line {
		c1 = col(r.End.Line, r.End.Character)
	}
	return c0, c1
}

// tryOpenDiskReadOnly 在工作区外以只读标签打开磁盘文件并跳到范围;force 跳过超大确认。
func (a *App) tryOpenDiskReadOnly(abs string, r lsp.Range, force bool) {
	abs = filepath.Clean(abs)
	st, err := os.Stat(abs)
	if err != nil {
		a.toastOpenFileErr(abs, err)
		return
	}
	if st.IsDir() {
		a.toast = "这是文件夹,不能用编辑器打开"
		return
	}
	if st.Size() > workspace.MaxRead {
		a.toast = "文件过大,不能用编辑器打开"
		return
	}
	open := func() {
		a.bg("ext.open", func() func() {
			defer diag.SpanSlow("ide", "读取工作区外文件", time.Second, "path", abs)()
			src, err := os.ReadFile(abs)
			return func() {
				if err != nil {
					a.toastOpenReadErr(abs, err)
					return
				}
				if strings.IndexByte(string(src), 0) >= 0 {
					a.toast = "这是二进制文件,不能用编辑器打开"
					return
				}
				a.finishOpenDiskReadOnly(abs, string(src), r)
			}
		})
	}
	if force || st.Size() <= externalOpenWarn {
		open()
		return
	}
	a.confirmDlg("文件较大,仍要打开?", filepath.Base(abs)+" · "+humanBytes(st.Size())+"\n只读打开,不会写入磁盘。", "仍然打开", false, open)
}

func (a *App) finishOpenDiskReadOnly(abs, src string, r lsp.Range) {
	d := a.ide
	if d == nil {
		return
	}
	c0, c1 := a.lspColsAt(abs, r, src)
	d.navMark()
	t := d.tab(abs)
	if t == nil {
		t = &ideTab{id: abs, path: abs, diskRO: true, preview: false}
		t.buf = editor.NewBuffer(src)
		t.buf.SetLanguage(editor.LanguageFor(filepath.Base(abs)))
		t.saved = src
		t.st = &editor.State{Buffer: t.buf, ReadOnly: true, WrapGuides: nil}
		d.addTab(t)
	} else {
		t.diskRO = true
		if t.st != nil {
			t.st.ReadOnly = true
		}
		if t.buf != nil && t.buf.String() != src && !t.dirty() {
			t.buf.Reset(src)
			t.saved = src
		}
	}
	t.reveal = &[3]int{r.Start.Line, c0, c1}
	d.sel = abs
	d.navPush(navPos{abs, r.Start.Line, c0})
	d.active = t.id
	a.SetMode(ModeIDE)
	a.revealWorkspace()
	extOpenLog.Info("只读打开工作区外", "path", abs, "line", r.Start.Line+1)
}

// jumpToExternalPos 从引用面板等跳转到工作区外的只读文件。
func (a *App) jumpToExternalPos(p string, line, col, end int) {
	p = filepath.Clean(p)
	build := func(ln string) lsp.Range {
		u0 := lsp.ColToUTF16(ln, col)
		u1 := u0
		if end > col {
			u1 = lsp.ColToUTF16(ln, end)
		}
		return lsp.Range{
			Start: lsp.Position{Line: line, Character: u0},
			End:   lsp.Position{Line: line, Character: u1},
		}
	}
	if t := a.ide.tab(p); t != nil && t.buf != nil {
		ln := ""
		if line >= 0 && line < t.buf.LineCount() {
			ln = t.buf.Line(line)
		}
		a.jumpToLSPAbs(p, build(ln))
		return
	}
	a.bg("ext.jump", func() func() {
		full, ln, err := readDiskTextLine(p, line)
		return func() {
			if err != nil {
				a.toastOpenFileErr(p, err)
				return
			}
			if strings.IndexByte(full, 0) >= 0 {
				a.toast = "这是二进制文件,不能用编辑器打开"
				return
			}
			st, err := os.Stat(p)
			if err != nil {
				a.toastOpenFileErr(p, err)
				return
			}
			if st.Size() > workspace.MaxRead {
				a.toast = "文件过大,不能用编辑器打开"
				return
			}
			open := func() { a.finishOpenDiskReadOnly(p, full, build(ln)) }
			if st.Size() > externalOpenWarn {
				a.confirmDlg("文件较大,仍要打开?", filepath.Base(p)+" · "+humanBytes(st.Size())+"\n只读打开,不会写入磁盘。", "仍然打开", false, open)
				return
			}
			open()
		}
	})
}

func readDiskTextLine(abs string, line int) (full, ln string, err error) {
	abs = filepath.Clean(abs)
	st, err := os.Stat(abs)
	if err != nil {
		return "", "", err
	}
	if st.Size() > workspace.MaxRead {
		return "", "", workspace.ErrTooLarge
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", "", err
	}
	full = string(b)
	lines := strings.Split(full, "\n")
	if line < 0 || line >= len(lines) {
		return full, "", nil
	}
	return full, lines[line], nil
}

// refLineTextBG 供引用列表在后台读工作区外文件的一行(调用方已在 goroutine 里)。
func (a *App) refLineTextBG(abs string, line int) (ln, body string) {
	_, ln, err := readDiskTextLine(abs, line)
	if err != nil {
		return "", "(无法读取)"
	}
	return ln, strings.TrimSpace(ln)
}
