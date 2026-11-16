package app

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

var eolLog = diag.L("ide-eol")

// 行尾序列(设计稿 ide-nav 三区清单「状态栏 · 编码 / 行尾」)。
// 编辑器缓冲区一律把 \r\n 归一成 \n;文件原来是 CRLF 的,保存时要还原,
// 否则「打开 → 保存」会悄悄把整个文件改成 LF(Git 里表现为每一行都变了)。
const (
	eolLF   = "\n"
	eolCRLF = "\r\n"
)

// detectEOL 取文件里占多数的行尾;没有换行 = LF。
func detectEOL(src string) string {
	crlf := strings.Count(src, "\r\n")
	if crlf > 0 && crlf*2 >= strings.Count(src, "\n") {
		return eolCRLF
	}
	return eolLF
}

// normEOL 与缓冲区一致:只有 \r\n 变成 \n(库不处理孤立的 \r,这里也不动)。
func normEOL(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

// withEOL 把已归一的文本写成目标行尾。
func withEOL(s, eol string) string {
	if eol != eolCRLF {
		return s
	}
	return strings.ReplaceAll(s, "\n", eolCRLF)
}

func eolLabel(eol string) string {
	if eol == eolCRLF {
		return "CRLF"
	}
	return "LF"
}

// eolMenu 是状态栏「LF / CRLF」点开的菜单:只改这个标签保存时用的行尾(文件内容里没有 \r,
// 所以切换后标签会显示为「未保存」,⌘S 之后才真正落盘)。
func (a *App) eolMenu(m *ui.Menu) {
	tb := a.ide.cur()
	if tb == nil || tb.buf == nil {
		return
	}
	ro := tb.st != nil && tb.st.ReadOnly
	for _, o := range []struct{ label, eol string }{{"LF(\\n,macOS / Linux)", eolLF}, {"CRLF(\\r\\n,Windows)", eolCRLF}} {
		it := menuItem(m, o.label).Radio(tb.eolNow() == o.eol)
		if ro {
			it.Disabled(true).Tooltip("只读文件不能改行尾")
		}
		if it.Chosen() && tb.eolNow() != o.eol {
			tb.eol = o.eol
			eolLog.Info("切换行尾", "path", tb.path, "eol", eolLabel(o.eol))
			a.toast = "行尾:" + eolLabel(o.eol) + "(保存后生效)"
		}
	}
}

// encMenu 是状态栏「UTF-8」点开的菜单。读写只支持 UTF-8,其余编码如实置灰并写明原因,
// 不假装能转换(打开非 UTF-8 文件会看到乱码,此时应在外部转码)。
func (a *App) encMenu(m *ui.Menu) {
	menuItem(m, "UTF-8").Radio(true)
	for _, n := range []string{"GBK", "UTF-16", "ISO-8859-1"} {
		menuItem(m, n).Disabled(true).Tooltip("暂不支持这种编码的读写,请先在外部转成 UTF-8")
	}
}
