package app

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 命令卡的语法高亮(设计稿 flow.css 的 .term / 语法色令牌):
//   - 命令行(shellSpans):命令名 / 子命令 / 参数标志 / 引号字符串 / 变量 / 管道与重定向 / 注释各一个颜色;
//   - 输出行(termLineSz):错误 / 警告 / 通过、URL、文件路径(可带 :行:列)、提交哈希、git 状态码、分支行……
// 颜色全部取自当前主题的语法色与状态色,换主题自动跟随。结果按「主题 + 字号 + 文本」缓存,
// 每帧只查表,不重复跑正则。

// hlCacheMax 是缓存条目上限,超过就整体清空(行文本千变万化,不值得做 LRU)。
const hlCacheMax = 6000

// hlLookup 取高亮缓存;主题 / 强调色 / 字号变了先清空。
func (a *App) hlLookup(kind string, size float32, text string) ([]ui.Span, string) {
	key := fmt.Sprintf("%s|%v|%v", a.pal.ID, a.tk.Ac, size)
	if a.hlKey != key || len(a.hlCache) > hlCacheMax {
		a.hlKey, a.hlCache = key, map[string][]ui.Span{}
	}
	ck := kind + "\x00" + text
	return a.hlCache[ck], ck
}

// ───────── 命令行 ─────────

var (
	// 带子命令的工具:第一个不以 - 开头的参数是子命令(git pull、cargo test)。
	shSubTools = map[string]bool{
		"git": true, "cargo": true, "npm": true, "pnpm": true, "yarn": true, "go": true, "docker": true,
		"kubectl": true, "brew": true, "pip": true, "pip3": true, "gh": true, "rustup": true, "bun": true,
		"deno": true, "systemctl": true, "apt": true, "apt-get": true, "helm": true, "terraform": true,
	}
	// 包装命令:后面跟的才是真正的命令。
	shWrappers = map[string]bool{"sudo": true, "env": true, "time": true, "nohup": true, "xargs": true, "exec": true, "command": true, "nice": true, "watch": true}

	shRedirect = regexp.MustCompile(`^\d*(?:>>?|<<?)&?\d*-?`)
	shVar      = regexp.MustCompile(`^\$(?:\{[^}]*\}|[A-Za-z_]\w*|[0-9@#?!*$-])`)
	shAssign   = regexp.MustCompile(`^([A-Za-z_]\w*)(=)`)
)

// shellSpans 把一行 shell 命令切成带色片段。多行命令只取第一行,后面补「 …」。
func (a *App) shellSpans(cmd string, size float32) []ui.Span {
	if i := strings.IndexByte(cmd, '\n'); i >= 0 {
		cmd = cmd[:i] + " …"
	}
	if sp, ck := a.hlLookup("sh", size, cmd); sp != nil {
		return sp
	} else {
		sp = a.shellTokens(cmd, size)
		a.hlCache[ck] = sp
		return sp
	}
}

func (a *App) shellTokens(cmd string, size float32) []ui.Span {
	t := a.tk
	var out []ui.Span
	emit := func(s string, col ui.Color, weight int, italic bool) {
		if s != "" {
			out = append(out, ui.Span{Text: s, Font: theme.CodeFont(), Size: size, Color: col, Weight: weight, Italic: italic})
		}
	}
	expect := true // 下一个词是命令名
	sub := false   // 下一个不以 - 开头的词是子命令
	i := 0
	for i < len(cmd) {
		rest := cmd[i:]
		c := cmd[i]
		switch {
		case c == ' ' || c == '\t':
			j := i
			for j < len(cmd) && (cmd[j] == ' ' || cmd[j] == '\t') {
				j++
			}
			emit(cmd[i:j], t.Tx1, 0, false)
			i = j
		case c == '#' && (i == 0 || cmd[i-1] == ' ' || cmd[i-1] == '\t'):
			emit(rest, t.SynCom, 0, true)
			i = len(cmd)
		case c == '"' || c == '\'':
			j := i + 1
			for j < len(cmd) && cmd[j] != c {
				if cmd[j] == '\\' && c == '"' && j+1 < len(cmd) {
					j++
				}
				j++
			}
			if j < len(cmd) {
				j++ // 含收尾引号
			}
			emit(cmd[i:j], t.SynStr, 0, false)
			i = j
			if expect {
				expect = false
			}
		case strings.HasPrefix(rest, "$("):
			emit("$(", t.Tx3, 0, false)
			i += 2
			expect, sub = true, false
		case c == '$' && shVar.MatchString(rest):
			m := shVar.FindString(rest)
			emit(m, t.SynNum, 0, false)
			i += len(m)
		case shRedirect.MatchString(rest) && shRedirect.FindString(rest) != "":
			m := shRedirect.FindString(rest)
			emit(m, t.Tx3, 0, false)
			i += len(m)
		case strings.IndexByte("|&;()", c) >= 0:
			j := i
			for j < len(cmd) && strings.IndexByte("|&;()", cmd[j]) >= 0 {
				j++
			}
			emit(cmd[i:j], t.Tx3, 600, false)
			i = j
			expect, sub = true, false
		default:
			j := i
			for j < len(cmd) && strings.IndexByte(" \t|&;<>()\"'", cmd[j]) < 0 {
				j++
			}
			if j == i {
				j = i + 1
			}
			w := cmd[i:j]
			i = j
			switch {
			case expect && shAssign.MatchString(w):
				m := shAssign.FindStringSubmatch(w)
				emit(m[1], t.SynNum, 0, false)
				emit(m[2], t.Tx3, 0, false)
				emit(w[len(m[0]):], t.Tx1, 0, false)
			case expect:
				emit(w, t.SynFn, 600, false)
				if shWrappers[w] {
					break // 仍在等真正的命令
				}
				expect = false
				sub = shSubTools[w]
			case strings.HasPrefix(w, "-") && len(w) > 1:
				name, val, has := strings.Cut(w, "=")
				emit(name, t.SynMac, 0, false)
				if has {
					emit("=", t.Tx3, 0, false)
					emit(val, t.Tx1, 0, false)
				}
			case sub:
				emit(w, t.SynKw, 600, false)
				sub = false
			case strings.HasPrefix(w, "http://") || strings.HasPrefix(w, "https://"):
				emit(w, t.AcTx, 0, false)
			case isAllDigits(w):
				emit(w, t.SynNum, 0, false)
			default:
				emit(w, t.Tx1, 0, false)
			}
		}
	}
	if len(out) == 0 {
		emit(" ", t.Tx1, 0, false)
	}
	return out
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// ───────── 输出行 ─────────

type hlRule struct {
	re     *regexp.Regexp
	col    func(a *App) ui.Color
	weight int
	italic bool
	// accept 非空时再校验一次匹配文本(正则表达不了的条件)。
	accept func(string) bool
}

var hlRules = []hlRule{
	// cargo / npm 等的阶段前缀(Compiling / Running…)
	{re: regexp.MustCompile(`^\s*(?:Compiling|Running|Finished|Downloaded|Checking|Fresh|Blocking|Updating)\b`), col: func(a *App) ui.Color { return a.tk.Tx3 }},
	// 错误 / 警告 / 提示前缀
	{re: regexp.MustCompile(`^\s*(?:error(?:\[\w+\])?|fatal|panic(?:ked)?|npm ERR!|ERROR|Error|FAIL(?:ED)?|--- FAIL)\b:?`), col: func(a *App) ui.Color { return a.tk.Err }, weight: 600},
	{re: regexp.MustCompile(`^\s*(?:warning|warn|WARN|WARNING|npm WARN)\b:?`), col: func(a *App) ui.Color { return a.tk.Warn }, weight: 600},
	{re: regexp.MustCompile(`^\s*(?:hint|note|help|remote|info):`), col: func(a *App) ui.Color { return a.tk.Tx3 }},
	// git 状态码(porcelain 短格式)与 `## 分支` 行
	{re: regexp.MustCompile(`^## .*$`), col: func(a *App) ui.Color { return a.tk.AcTx }},
	{re: regexp.MustCompile(`^(?:UU|AA|DD|AU|UA|DU|UD)(?: )`), col: func(a *App) ui.Color { return a.tk.Err }, weight: 600},
	{re: regexp.MustCompile(`^(?:[ M]M|M[ M])(?: )`), col: func(a *App) ui.Color { return a.tk.Warn }, weight: 600},
	{re: regexp.MustCompile(`^A[ M] `), col: func(a *App) ui.Color { return a.tk.Ok }, weight: 600},
	{re: regexp.MustCompile(`^\?\? `), col: func(a *App) ui.Color { return a.tk.Tx3 }, weight: 600},
	{re: regexp.MustCompile(`^(?:[ D]D|D )(?: )`), col: func(a *App) ui.Color { return a.tk.Err }, weight: 600},
	// URL
	{re: regexp.MustCompile(`https?://[^\s)\]'"<>]+`), col: func(a *App) ui.Color { return a.tk.AcTx }},
	// 双引号 / 反引号字符串
	{re: regexp.MustCompile("\"[^\"\\n]*\"|`[^`\\n]*`"), col: func(a *App) ui.Color { return a.tk.SynStr }},
	// 通过 / 失败 / 警告的关键词
	{re: regexp.MustCompile(`\b(?:ok|OK|PASS|PASSED|passed|Successfully|succeeded|success|Done|done)\b`), col: func(a *App) ui.Color { return a.tk.Ok }},
	{re: regexp.MustCompile(`\b(?:FAILED|failed|CONFLICT|conflicts?|denied|refused|error|Error|ERROR|fatal|No such file or directory|not found)\b`), col: func(a *App) ui.Color { return a.tk.Err }},
	{re: regexp.MustCompile(`\b(?:warning|warnings|WARN|WARNING|deprecated|Deprecated)\b`), col: func(a *App) ui.Color { return a.tk.Warn }},
	// 文件路径(可带 :行:列):含目录的路径,或带常见扩展名的文件名;以数字开头的(如 1/1)不算
	{re: regexp.MustCompile(`(?:[A-Za-z_.~][\w.~-]*/)+[\w.-]*[A-Za-z_][\w.-]*(?::\d+(?::\d+)?)?|\b[\w-]+\.(?:go|rs|ts|tsx|js|jsx|py|md|json|toml|lock|yml|yaml|sh|css|html|swift|c|h|cpp|txt)\b(?::\d+(?::\d+)?)?`),
		col: func(a *App) ui.Color { return a.tk.Info }},
	// 提交哈希(7–40 位十六进制,要同时含数字与 a-f 字母,免得把纯数字当哈希)
	{re: regexp.MustCompile(`\b[0-9a-f]{7,40}\b`), col: func(a *App) ui.Color { return a.tk.SynNum }, accept: looksLikeHash},
	// 用时 / 箭头 / 带单位的数字
	{re: regexp.MustCompile(`\bin \d+(?:\.\d+)?(?:ms|s)\b`), col: func(a *App) ui.Color { return a.tk.Tx3 }},
	{re: regexp.MustCompile(`->|→|\.\.\.`), col: func(a *App) ui.Color { return a.tk.Tx3 }},
	{re: regexp.MustCompile(`\b\d+(?:\.\d+)?(?:ms|KB|MB|GB|%)(?:\b|$)`), col: func(a *App) ui.Color { return a.tk.SynNum }},
}

func looksLikeHash(s string) bool {
	digit, letter := false, false
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digit = true
		} else {
			letter = true
		}
	}
	return digit && letter
}

type hlMark struct {
	i, j   int
	col    ui.Color
	weight int
	italic bool
}

// termLine 把一行终端输出切成带色片段(11.5 号;IDE / 工作区面板用)。
func (a *App) termLine(line string) []ui.Span { return a.termLineSz(line, 11.5) }

// termLineSz 同 termLine,字号可指定(会话流跟随「会话字号」缩放)。
func (a *App) termLineSz(line string, size float32) []ui.Span {
	if sp, ck := a.hlLookup("out", size, line); sp != nil {
		return sp
	} else {
		sp = a.termTokens(line, size)
		a.hlCache[ck] = sp
		return sp
	}
}

func (a *App) termTokens(line string, size float32) []ui.Span {
	t := a.tk
	mk := func(s string, col ui.Color, weight int, italic bool) ui.Span {
		return ui.Span{Text: s, Font: theme.CodeFont(), Size: size, Color: col, Weight: weight, Italic: italic}
	}
	// 超长行(进度条、压缩后的单行 JSON)不跑正则,免得每帧白白耗时。
	if len(line) > 600 {
		return []ui.Span{mk(line, t.Tx1, 0, false)}
	}
	var ms []hlMark
	overlap := func(i, j int) bool {
		for _, m := range ms {
			if i < m.j && j > m.i {
				return true
			}
		}
		return false
	}
	for _, r := range hlRules {
		for _, loc := range r.re.FindAllStringIndex(line, 16) {
			if loc[1] <= loc[0] || overlap(loc[0], loc[1]) {
				continue
			}
			if r.accept != nil && !r.accept(line[loc[0]:loc[1]]) {
				continue
			}
			ms = append(ms, hlMark{loc[0], loc[1], r.col(a), r.weight, r.italic})
		}
		if len(ms) > 64 {
			break
		}
	}
	sort.Slice(ms, func(x, y int) bool { return ms[x].i < ms[y].i })
	var out []ui.Span
	last := 0
	for _, m := range ms {
		if m.i > last {
			out = append(out, mk(line[last:m.i], t.Tx1, 0, false))
		}
		out = append(out, mk(line[m.i:m.j], m.col, m.weight, m.italic))
		last = m.j
	}
	if last < len(line) {
		out = append(out, mk(line[last:], t.Tx1, 0, false))
	}
	if len(out) == 0 {
		out = append(out, mk(" ", t.Tx1, 0, false))
	}
	return out
}
