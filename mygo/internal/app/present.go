package app

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
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
	imgRead    bool // 读取的是本机存在的图片:展开体里直接显示这张图(flow_read_image.go)
	plainObj   bool // 对象是一句话(如联网搜索的查询词):与动词同色(Tx3),不按「对象」提亮

	// 行内动作(悬停出现,点对象名 / 图标也能跳):
	path    string              // 涉及的文件(原样,可能是绝对路径),空 = 无
	line    int                 // 跳转行(1 起,0 = 不定位)
	diffAs  bool                // 有文件差异可看(编辑 / 写入)
	term    *core.TerminalBlock // 命令输出(可在终端面板查看)
	url     string              // 抓取的网址
	copyStr string              // 「复制」的内容(命令 / 路径 / 网址 / 搜索词)

	thinkText string // 思考正文(跑马灯预览)
	thinkLive bool   // 思考流式中
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
			a, r := reviewStat(d)
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
		// 读了哪几行:起点 = offset(缺省从第 1 行),行数 = 结果的行数(给了 limit 且结果更长时以 limit 为准);
		// 还没有结果时只有 limit 才知道终点。读取行不展开(没有可看的内容,点整行 = 在工作区打开并定位到起点)。
		start := max(inputInt(tc, "offset"), 1)
		s.line = start
		s.meta = dirOf(a.shortPath(path))
		n := textLines(firstText(tc))
		if lim := inputInt(tc, "limit"); lim > 0 && (n == 0 || n > lim) {
			n = lim
		}
		if n > 0 {
			s.meta += fmt.Sprintf(" · L%d–%d", start, start+n-1)
			s.right = fmt.Sprintf("%d 行", n)
		} else if inputInt(tc, "offset") > 0 {
			s.meta += fmt.Sprintf(" · L%d 起", start)
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
			unit := "处" // 设计稿:Grep「5 处」,Glob「3 个文件」
			if tc.Name == "Glob" {
				unit = "个文件"
			}
			s.right = fmt.Sprintf("%d %s", n, unit)
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
	case core.KindExecute:
		s.fam = famExec
		s.icon, s.verb = "terminal", "运行"
		s.obj = inputStr(tc, "command")
		if s.obj == "" {
			s.obj = tc.Title
		}
		s.right = durText(tc.DurationMs)
		a.execTokens(&s, tc)
		s.copyStr = s.obj
		for _, ct := range tc.Content {
			if tb, ok := ct.(*core.TerminalBlock); ok {
				s.term = tb
			}
		}
		s.hasBody = hasContent[*core.TerminalBlock](tc)
	case core.KindFetch:
		s.fam = famWeb
		s.icon, s.verb, s.obj = "globe", "抓取", firstNonEmpty(inputStr(tc, "url"), inputStr(tc, "query"), tc.Title)
		if tc.Name == "WebSearch" { // 设计稿:联网搜索 <查询词>,右侧「N 条结果」;查询词与动词同色同字体(plainObj)
			s.verb = "联网搜索"
			s.plainObj = true
			if n := strings.Count(firstText(tc), `"url"`); n > 0 {
				s.right = fmt.Sprintf("%d 条结果", n)
			}
		}
		s.copyStr = s.obj
		if u := inputStr(tc, "url"); strings.HasPrefix(u, "http") {
			s.url = u
		}
	case core.KindMCP, core.KindBrowser:
		s.fam = famMCP
		s.icon, s.verb, s.obj = "plug", "调用", tc.Title
	default:
		// 其余工具统一扳手图标(不是文档图标);技能直接写「技能 名称」,不再多一层「工具」
		s.icon, s.verb, s.obj = "wrench", "工具", tc.Title
		if tc.Name == "Skill" {
			s.icon, s.verb, s.obj = "wand", "技能", strings.TrimPrefix(tc.Title, "技能 · ")
		}
	}
	switch s.state {
	case "err":
		s.icon = "errc"
		s.right = failedRight(tc, s.right)
	case "deny":
		s.icon, s.right = "lock", "已拒绝" // 设计稿:图标是锁、对象带删除线、右侧「已拒绝」
	}
	// 读取成功不展开(设计稿:读取行没有展开箭头,只写「读取 文件 路径 · L起–止」);失败 / 被拒绝仍可展开看原因
	readPlain := tc.Kind == core.KindRead && s.state != "err" && s.state != "deny"
	if !s.hasBody && !readPlain && toolHasDetail(tc) { // MCP / 浏览器 / 搜索 / 抓取…:展开看入参与结果
		s.hasBody = true
	}
	a.applyToolDefaultOpen(&s, tc)
	if a.readImagePath(tc) != "" { // 读图:行下面直接显示图(默认展开;精简档收起),点图片开灯箱
		s.hasBody, s.imgRead = true, true
		s.open = a.ToolDetailLevel() != 0
	}
	return s
}

// applyToolDefaultOpen 按「工具调用详细程度」与设计稿 flow-detail 默认展开矩阵设置 sp.open(用户手动折叠记在 folds,stepRow 会覆盖)。
func (a *App) applyToolDefaultOpen(s *step, tc *core.ToolCall) {
	if !s.hasBody {
		return
	}
	lv := a.ToolDetailLevel()
	switch lv {
	case 0:
		s.open = false
		return
	case 2:
		s.open = tc.Status == core.ToolCompleted || tc.Status == core.ToolRunning || tc.Status == core.ToolPending
		if s.state == "deny" { // 失败的不自动展开(只在行尾标红),被拒绝的仍展开
			s.open = true
		}
		return
	}
	// 标准档:默认折叠;被拒绝 / 运行中的命令 / 非 0 退出码展开(失败的不自动展开)
	s.open = false
	if s.state == "deny" {
		s.open = true
		return
	}
	if tc.Kind == core.KindExecute {
		if s.state == "run" {
			s.open = true
			// 设计稿:运行中展开实时输出,结束后不自动收起(避免正在看的内容突然缩回、下面的行跳动)。
			// 记进手动状态;用户在运行期间自己收起过的(folds 里已有记录)不覆盖。
			if _, set := a.folds[s.id]; !set {
				a.folds[s.id] = true
			}
			return
		}
		for _, ct := range tc.Content {
			if tb, ok := ct.(*core.TerminalBlock); ok && tb.ExitCode != nil && *tb.ExitCode != 0 {
				s.open = true
				return
			}
		}
	}
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// thinkTitle 取思考内容的第一句作标题(流式尚无正文时写「思考中」)。
func thinkTitle(text string) string {
	if t := thinkFirstLine(text); t != "" { // 不按字数截断:行放不下时由文字控件按宽度省略
		return t
	}
	return "思考中"
}

// isExplore 探索类工具(读 / 搜 / 查找),连续的两个以上会收成一个分组行。
func isExplore(it core.Item) (*core.ToolCall, bool) {
	tc := doneToolOf(it) // 已处理的权限请求里装的也是工具调用,不能把一串探索从中间截断
	if tc == nil {
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

// execTokens 命令类工具的令牌(设计稿 flow 页「工具令牌」):后台 Bash = 「后台运行」+ 活动图标 + 右侧「已启动」,
// Monitor = 「监听」+ 眼睛图标,后台输出 / 结束后台任务各有自己的动词,不再都叫「运行」。
func (a *App) execTokens(s *step, tc *core.ToolCall) {
	switch tc.Name {
	case "Monitor":
		s.icon, s.verb = "eye", "监听"
		s.obj = firstNonEmpty(inputStr(tc, "description"), inputStr(tc, "command"), strings.TrimPrefix(tc.Title, "监听 "))
	case "BashOutput", "TaskOutput":
		s.icon, s.verb = "activity", "后台输出"
		s.obj = firstNonEmpty(tc.TaskID, strings.TrimPrefix(tc.Title, "读取后台输出"))
	case "KillShell", "KillBash", "TaskStop":
		s.icon, s.verb = "stop", "结束后台任务"
		s.obj = tc.TaskID
	default:
		if bg, _ := tc.Input["run_in_background"].(bool); bg {
			s.icon, s.verb = "activity", "后台运行"
			if tc.Status == core.ToolCompleted {
				s.right = "已启动"
			}
		}
	}
}

// failedRight 失败 / 被中断的步骤右侧写什么:命令给退出码(设计稿「退出码 101」,红字),被中断写「已中断」,其余沿用原右侧。
func failedRight(tc *core.ToolCall, cur string) string {
	if tc.Status == core.ToolInterrupted {
		return "已中断"
	}
	for _, ct := range tc.Content {
		if tb, ok := ct.(*core.TerminalBlock); ok && tb.ExitCode != nil && *tb.ExitCode != 0 {
			if deniedOutput(tb.Output) {
				// 被用户拒绝的命令根本没跑过,「退出码 1」只是拒绝信号
				return "未执行"
			}
			return "退出码 " + strconv.Itoa(*tb.ExitCode)
		}
	}
	return cur
}

// inputInt 读入参里的整数(JSON 数字解出来是 float64;也兼容整型与数字字符串);没有或读不出返回 0。
func inputInt(tc *core.ToolCall, k string) int {
	switch v := tc.Input[k].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(v)
		return n
	}
	return 0
}
