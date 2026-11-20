package app

import (
	"path"
	"strconv"
	"strings"
)

// parsePathLine 从命令面板查询解析「路径:行[:列]」。相对路径无盘符,只认最后一个「:数字」后缀。
func parsePathLine(q string) (p string, line, col int, ok bool) {
	q = strings.TrimSpace(q)
	if q == "" {
		return "", 0, 0, false
	}
	if strings.HasPrefix(q, ":") {
		rest := strings.TrimSpace(q[1:])
		if n, c, ok2 := parseLineCol(rest); ok2 {
			return "", n, c, true
		}
		return "", 0, 0, false
	}
	for i := len(q) - 1; i >= 0; i-- {
		if q[i] != ':' {
			continue
		}
		tail := strings.TrimSpace(q[i+1:])
		head := strings.TrimSpace(q[:i])
		line, col, ok2 := parseLineCol(tail)
		if !ok2 || head == "" {
			continue
		}
		// 「file.rs:行:列」:末尾单独数字时,再试「路径 + 行:列」两段解析
		if col == 0 && !strings.Contains(tail, ":") {
			if j := strings.LastIndex(head, ":"); j >= 0 {
				mid := strings.TrimSpace(head[j+1:]) + ":" + tail
				if n, c, ok3 := parseLineCol(mid); ok3 {
					p = strings.TrimSpace(head[:j])
					if p != "" {
						return p, n, c, true
					}
				}
			}
		}
		return head, line, col, true
	}
	return "", 0, 0, false
}

func parseLineCol(s string) (line, col int, ok bool) {
	if s == "" {
		return 0, 0, false
	}
	parts := strings.SplitN(s, ":", 2)
	n, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || n < 1 {
		return 0, 0, false
	}
	line = n - 1
	col = 0
	if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
		c, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil || c < 1 {
			return 0, 0, false
		}
		col = c - 1
	}
	return line, col, true
}

// openFileAtLine 打开文件并跳到行(行号 0 起);列与行尾用于选中。
// 不切换形态:在 Agent 形态就开在右侧工作区的编辑器里,在 IDE 形态就开在编辑区(会话里点文件走这里,见设计稿 arch.js「会话流里的文件链接」)。
// 命令面板要「打开后进 IDE」的,由调用方自己 SetMode。
func (a *App) openFileAtLine(p string, line, col int) {
	a.ide.openAt(p, line, col, col, true)
	a.ide.sel = p
}

// gotoLineInCurrent 转到当前标签的某行与列(0 起);超界夹到末行,列夹到行尾。
func (a *App) gotoLineInCurrent(line, col int) {
	tb := a.ide.cur()
	if tb == nil || tb.st == nil || tb.buf == nil || tb.diff || tb.virt != "" {
		a.toast = "先打开一个文件"
		return
	}
	n := tb.buf.LineCount()
	clamped, ok := a.jumpToLineCol(tb, line, col)
	if !ok {
		return
	}
	if clamped {
		a.toast = "共 " + itoa(n) + " 行,已跳到末行"
	}
}

// paletteGotoLine 处理命令面板里的「:行号」或带路径的 :行号打开项。
func (a *App) paletteGotoLine(q string) bool {
	p, line, col, ok := parsePathLine(q)
	if !ok {
		return false
	}
	if p == "" {
		tb := a.ide.cur()
		if tb == nil || tb.buf == nil || tb.diff {
			a.toast = "先打开要编辑的文件"
			return true
		}
		a.gotoLineInCurrent(line, col)
		return true
	}
	a.openFileAtLine(p, line, col)
	a.SetMode(ModeIDE) // 命令面板打开文件仍进 IDE 形态(openFileAtLine 本身不切)
	return true
}

// paletteOpenFile 命令面板选中文件项:支持查询里带 :行号。
func (a *App) paletteOpenFile(p, query string) {
	if i := strings.LastIndex(query, ":"); i > 0 {
		head, tail := strings.TrimSpace(query[:i]), strings.TrimSpace(query[i+1:])
		if line, col, ok := parseLineCol(tail); ok {
			if head == p || fuzzy(head, p) >= 0 || fuzzy(head, path.Base(p)) >= 0 || strings.HasSuffix(p, head) {
				a.openFileAtLine(p, line, col)
				a.SetMode(ModeIDE)
				return
			}
		}
	}
	a.ide.openFile(p, true)
	a.SetMode(ModeIDE)
}
