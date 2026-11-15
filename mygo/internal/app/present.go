package app

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diff"
)

// 工具族(对应设计稿 data-tool 的色相):只染图标。
type family int

const (
	famExplore family = iota
	famEdit
	famExec
	famWeb
	famMCP
	famAgent
	famPlan
	famThink
)

func (a *App) famColor(f family) ui.Color {
	t := a.tk
	switch f {
	case famExplore:
		return t.ToolExplore
	case famEdit:
		return t.ToolEdit
	case famExec:
		return t.ToolExec
	case famWeb:
		return t.ToolWeb
	case famMCP:
		return t.ToolMCP
	case famAgent:
		return t.ToolAgent
	case famPlan:
		return t.ToolPlan
	}
	return t.ToolThink
}

// step 是会话流里一行「步骤」的展示数据(由工具调用 / 思考等条目翻译而来)。
type step struct {
	id         string
	icon, verb string
	obj, meta  string
	right      string // 右侧结果(纯文字);差异增删另用 add / del
	add, del   int
	fam        family
	state      string // "" 完成 / run 运行中 / err 失败 / deny 已拒绝
	open       bool   // 默认展开
	hasBody    bool

	// 行内动作(悬停出现,点对象名 / 图标也能跳):
	path    string              // 涉及的文件(原样,可能是绝对路径),空 = 无
	line    int                 // 跳转行(1 起,0 = 不定位)
	diffAs  bool                // 有文件差异可看(编辑 / 写入)
	term    *core.TerminalBlock // 命令输出(可在终端面板查看)
	url     string              // 抓取的网址
	copyStr string              // 「复制」的内容(命令 / 路径 / 网址 / 搜索词)
}

func baseName(p string) string { return filepath.Base(p) }

func dirOf(p string) string {
	d := filepath.Dir(p)
	if d == "." || d == "/" {
		return ""
	}
	return d + "/"
}

func inputStr(tc *core.ToolCall, k string) string {
	if s, ok := tc.Input[k].(string); ok {
		return s
	}
	return ""
}

// textLines 数一段文字的行数。
func textLines(s string) int {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

// firstText 取工具结果里的第一段文本。
func firstText(tc *core.ToolCall) string {
	for _, c := range tc.Content {
		if tb, ok := c.(*core.TextBlock); ok {
			return tb.Text
		}
	}
	return ""
}

// diffStat 统计一个工具调用所有 Diff 块的增删行数。
func diffStat(tc *core.ToolCall) (add, del int) {
	for _, c := range tc.Content {
		if d, ok := c.(*core.DiffBlock); ok {
			a, r := diff.Stat(diff.Lines(d.Old, d.New, d.Line, d.Line))
			add, del = add+a, del+r
		}
	}
	return
}

func hasContent[T core.ToolContent](tc *core.ToolCall) bool {
	for _, c := range tc.Content {
		if _, ok := c.(T); ok {
			return true
		}
	}
	return false
}

func durText(ms int64) string {
	if ms <= 0 {
		return ""
	}
	if ms < 1000 {
		return fmt.Sprintf("%d 毫秒", ms)
	}
	if ms < 10_000 {
		return fmt.Sprintf("%.1f 秒", float64(ms)/1000)
	}
	return core.FormatElapsed(ms)
}

// presentTool 把一个工具调用翻译成步骤行。
func (a *App) presentTool(tc *core.ToolCall) step {
	s := step{id: tc.ID, fam: famExplore}
	switch tc.Status {
	case core.ToolRunning, core.ToolPending:
		s.state = "run"
	case core.ToolFailed, core.ToolInterrupted:
		s.state = "err"
	case core.ToolDenied:
		s.state = "deny"
	}
	path := inputStr(tc, "file_path")
	if path == "" && len(tc.Locations) > 0 {
		path = tc.Locations[0]
	}
	switch tc.Kind {
	case core.KindRead:
		s.icon, s.verb, s.obj = "fileText", "读取", baseName(path)
		s.path, s.copyStr = path, path
		s.line, _ = strconv.Atoi(inputStr(tc, "offset"))
		s.meta = dirOf(a.shortPath(path))
		if n := textLines(firstText(tc)); n > 0 {
			s.meta += fmt.Sprintf(" · L1–%d", n)
			s.right = fmt.Sprintf("%d 行", n)
		}
	case core.KindSearch:
		pat := inputStr(tc, "pattern")
		if tc.Name == "Glob" {
			s.icon, s.verb = "folder", "查找文件"
		} else {
			s.icon, s.verb = "search", "搜索"
		}
		s.obj = `"` + pat + `"`
		s.copyStr = pat
		s.meta = inputStr(tc, "path")
		txt := firstText(tc)
		if strings.HasPrefix(txt, "No ") || (tc.Status == core.ToolCompleted && txt == "") {
			s.right = "无匹配"
		} else if n := textLines(txt); n > 0 {
			if strings.HasPrefix(txt, "Found ") {
				n--
			}
			s.right = fmt.Sprintf("%d 处", n)
		}
	case core.KindEdit, core.KindWrite:
		s.fam = famEdit
		s.icon, s.verb, s.obj, s.meta = "edit", "编辑", baseName(path), dirOf(a.shortPath(path))
		if tc.Kind == core.KindWrite {
			s.icon, s.verb = "file", "写入"
		}
		s.path, s.copyStr, s.diffAs = path, path, true
		s.add, s.del = diffStat(tc)
		s.hasBody = hasContent[*core.DiffBlock](tc)
		s.open = s.hasBody && tc.Status == core.ToolCompleted && s.add+s.del <= 8 // 小改动默认展开,大改动折叠
	case core.KindExecute:
		s.fam = famExec
		s.icon, s.verb = "terminal", "运行"
		s.obj = inputStr(tc, "command")
		if s.obj == "" {
			s.obj = tc.Title
		}
		s.right = durText(tc.DurationMs)
		s.copyStr = s.obj
		for _, ct := range tc.Content {
			if tb, ok := ct.(*core.TerminalBlock); ok {
				s.term = tb
			}
		}
		s.hasBody = hasContent[*core.TerminalBlock](tc)
		s.open = s.hasBody
	case core.KindFetch:
		s.fam = famWeb
		s.icon, s.verb, s.obj = "globe", "抓取", firstNonEmpty(inputStr(tc, "url"), inputStr(tc, "query"), tc.Title)
		s.copyStr = s.obj
		if u := inputStr(tc, "url"); strings.HasPrefix(u, "http") {
			s.url = u
		}
	case core.KindMCP, core.KindBrowser:
		s.fam = famMCP
		s.icon, s.verb, s.obj = "plug", "调用", tc.Title
	default:
		s.icon, s.verb, s.obj = "fileText", "工具", tc.Title
	}
	switch s.state {
	case "err":
		s.icon = "errc"
	case "deny":
		s.icon = "lock"
	}
	if s.hasBody {
		switch a.ToolDetailLevel() {
		case 0:
			s.open = false
		case 2:
			if tc.Status == core.ToolCompleted || tc.Status == core.ToolRunning || tc.Status == core.ToolPending {
				s.open = true
			}
		}
	}
	return s
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// thinkTitle 取思考内容的第一句作标题(没有就叫「思考」)。
func thinkTitle(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if i := strings.IndexAny(text, "。.\n"); i > 0 {
		text = text[:i]
	}
	r := []rune(text)
	if len(r) > 28 {
		text = string(r[:28]) + "…"
	}
	return text
}

// isExplore 探索类工具(读 / 搜 / 查找),连续的两个以上会收成一个分组行。
func isExplore(it core.Item) (*core.ToolCall, bool) {
	tc, ok := it.(*core.ToolCall)
	if !ok {
		return nil, false
	}
	return tc, tc.Kind == core.KindRead || tc.Kind == core.KindSearch
}

// exploreSummary 「2 个文件 · 1 次搜索」。
func exploreSummary(g []*core.ToolCall) (string, bool) {
	files, searches, running := 0, 0, false
	for _, tc := range g {
		if tc.Kind == core.KindRead {
			files++
		} else {
			searches++
		}
		if !tc.Status.Finished() {
			running = true
		}
	}
	var parts []string
	if files > 0 {
		parts = append(parts, fmt.Sprintf("%d 个文件", files))
	}
	if searches > 0 {
		parts = append(parts, fmt.Sprintf("%d 次搜索", searches))
	}
	return strings.Join(parts, " · "), running
}
