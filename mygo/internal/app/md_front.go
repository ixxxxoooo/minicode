package app

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// Markdown 文件开头的 YAML 头(SKILL.md / .mdc 规则 / 带元数据的文档)不当正文渲染:
// 按普通 Markdown 渲染会把 `---` 画成分隔线、把头里的最后一行连同下一条 `---` 变成一行大标题(setext 标题)。
// 这里把它解析成「属性」卡片放在正文上方(设计稿 ide-md「YAML 头:属性卡片」);解析不了的头整段退回等宽源码,不丢内容。

// fmKind 是一条属性值的呈现方式。
type fmKind uint8

const (
	fmText fmKind = iota // 文本:折行显示
	fmList               // 清单:画成胶囊
	fmBool               // true / false:画成状态标
)

// fmEntry 是 YAML 头里的一条顶层属性。
type fmEntry struct {
	Key  string
	Kind fmKind
	Text string   // fmText / fmBool 的值
	List []string // fmList 的各项
}

var (
	fmKeyRe   = regexp.MustCompile(`^([A-Za-z0-9_][A-Za-z0-9_.\-]*)\s*:(?:\s+(.*))?$`)
	fmBlockRe = regexp.MustCompile(`^[>|][+\-]?\d*$`)
	// fmListKeys 是常见的「逗号分隔也算清单」的键(Claude Code 的技能 / 规则 / 子 Agent 头里常见)。
	fmListKeys = map[string]bool{
		"allowed-tools": true, "allowedtools": true, "disallowed-tools": true, "tools": true,
		"globs": true, "paths": true, "tags": true, "aliases": true, "keywords": true,
	}
)

// fmIndented 是不是缩进行(YAML 里属于上一个键的续行)。
func fmIndented(s string) bool { return s != "" && (s[0] == ' ' || s[0] == '\t') }

// fmUnquote 去掉成对的引号。
func fmUnquote(s string) string {
	s = strings.TrimSpace(s)
	if n := len(s); n >= 2 && (s[0] == '"' && s[n-1] == '"' || s[0] == '\'' && s[n-1] == '\'') {
		return s[1 : n-1]
	}
	return s
}

// fmSplitList 把 `a, b, "c"` 拆成清单项(空项丢弃)。
func fmSplitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = fmUnquote(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// parseFrontEntries 解析 YAML 头(带或不带首尾的 `---` 行),只认顶层 `键: 值`:
// 块标量(> / |)、缩进续行、`- ` 清单、`[a, b]` 行内清单。遇到没法归到某个键下的行就返回 ok=false,
// 调用方整段退回等宽源码。
func parseFrontEntries(front string) (out []fmEntry, ok bool) {
	front = strings.ReplaceAll(front, "\r\n", "\n")
	lines := strings.Split(strings.TrimRight(front, "\n"), "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		lines = lines[1:]
	}
	if n := len(lines); n > 0 && strings.TrimSpace(lines[n-1]) == "---" {
		lines = lines[:n-1]
	}
	for i := 0; i < len(lines); {
		ln := strings.TrimRight(lines[i], " \t")
		i++
		if tr := strings.TrimSpace(ln); tr == "" || strings.HasPrefix(tr, "#") {
			continue
		}
		if fmIndented(ln) {
			return nil, false // 没有键可挂的缩进行
		}
		m := fmKeyRe.FindStringSubmatch(ln)
		if m == nil {
			return nil, false
		}
		key, val := m[1], strings.TrimSpace(m[2])

		// 收集续行:缩进行、块内空行,以及值为空时顶格的 `- ` 清单项
		var cont []string
		for i < len(lines) {
			nx := strings.TrimRight(lines[i], " \t")
			if nx == "" { // 空行:后面还有缩进续行才算块内
				j := i
				for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
					j++
				}
				if j < len(lines) && fmIndented(lines[j]) {
					cont = append(cont, "")
					i++
					continue
				}
				break
			}
			if fmIndented(nx) || val == "" && (strings.HasPrefix(nx, "- ") || nx == "-") {
				cont = append(cont, strings.TrimSpace(nx))
				i++
				continue
			}
			break
		}
		for len(cont) > 0 && cont[len(cont)-1] == "" {
			cont = cont[:len(cont)-1]
		}
		out = append(out, fmBuild(key, val, cont))
	}
	return out, len(out) > 0
}

// fmBuild 按值的写法决定一条属性怎么呈现。
func fmBuild(key, val string, cont []string) fmEntry {
	e := fmEntry{Key: key}
	switch {
	case val == "" && len(cont) > 0 && fmAllDash(cont): // 块清单
		for _, c := range cont {
			if c = fmUnquote(strings.TrimPrefix(strings.TrimPrefix(c, "-"), " ")); c != "" {
				e.List = append(e.List, c)
			}
		}
		e.Kind = fmList
		return e
	case fmBlockRe.MatchString(val): // 块标量:> 折成一段(空行 = 段落),| 保留换行
		sep := "\n"
		if val[0] == '>' {
			sep = " "
		}
		var b strings.Builder
		for k, c := range cont {
			switch {
			case c == "":
				b.WriteString("\n")
			case k > 0 && cont[k-1] != "":
				b.WriteString(sep + c)
			default:
				b.WriteString(c)
			}
		}
		e.Text = b.String()
		return e
	case val == "" && len(cont) > 0: // 嵌套映射等:逐行原样给文本
		e.Text = strings.Join(cont, "\n")
		return e
	}
	if len(cont) > 0 { // 普通标量跨多行:折成一段
		val += " " + strings.Join(cont, " ")
	}
	if n := len(val); n >= 2 && val[0] == '[' && val[n-1] == ']' {
		e.Kind, e.List = fmList, fmSplitList(val[1:n-1])
		return e
	}
	val = fmUnquote(val)
	switch strings.ToLower(val) {
	case "true", "false":
		e.Kind, e.Text = fmBool, strings.ToLower(val)
		return e
	}
	if fmListKeys[strings.ToLower(key)] && strings.Contains(val, ",") {
		e.Kind, e.List = fmList, fmSplitList(val)
		return e
	}
	e.Text = val
	return e
}

func fmAllDash(cont []string) bool {
	for _, c := range cont {
		if c != "-" && !strings.HasPrefix(c, "- ") {
			return false
		}
	}
	return true
}

// mdFrontCard 画 YAML 头的「属性」卡片(只读)。id 区分折叠状态(一般用文件路径);
// maxH > 0 时卡片身体超过这个高度就在卡片内滚动(固定在编辑器顶部的用法,免得长描述把正文挤没)。
func (a *App) mdFrontCard(c *ui.Context, id, front string, maxH float32) {
	if strings.TrimSpace(front) == "" {
		return
	}
	t := a.tk
	ents, ok := parseFrontEntries(front)
	folded := a.ux.mdFrontOff[id]
	n := len(ents)

	box := ui.Column(c).Key("mdfront-" + id).Shrink(0).Radius(t.RMd()).Background(t.Fill1).BorderWidth(1).BorderColor(t.Bd1)
	if maxH > 0 { // 固定在编辑器顶部:左右与正文的 34 对齐
		box.Margin(12, 34, 4, 34)
	} else { // 嵌在别的滚动区里:只在下面留一点空
		box.Margin(0, 0, 10, 0)
	}
	box.Children(func() {
		head := ui.Row(c).Key("mdfront-h-"+id).Padding(5, 10).Gap(6).AlignItems(ui.Center).Cursor(ui.CursorPointer)
		if !folded {
			head.BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1)
		}
		head.OnClick(func() {
			if a.ux.mdFrontOff == nil {
				a.ux.mdFrontOff = map[string]bool{}
			}
			if folded {
				delete(a.ux.mdFrontOff, id)
			} else {
				a.ux.mdFrontOff[id] = true
			}
		})
		head.Children(func() {
			kit.Ic(c, map[bool]string{true: "chevR", false: "chevD"}[folded], kit.S12).TextColor(t.Tx4).Shrink(0)
			ui.Text(c, "属性").FontSize(theme.FsMicro).FontWeight(550).TextColor(t.Tx3).Shrink(0)
			if ok {
				ui.Text(c, itoa(n)+" 项").FontSize(theme.FsMicro).TextColor(t.Tx4).Shrink(0)
			}
			if folded && ok { // 折叠时留一行摘要:name 优先,否则第一项
				ui.Text(c, fmSummary(ents)).FontSize(theme.FsMicro).TextColor(t.Tx3).SingleLine().Grow(1).MinWidth(0)
			}
		})
		if folded {
			return
		}
		body := func() {
			if !ok { // 解析不了:整段等宽源码
				kit.Mono(c, strings.TrimSpace(fmStrip(front)), t.Tx2).LineHeight(1.5).Padding(8, 10)
				return
			}
			kw := fmKeyWidth(ents)
			ui.Column(c).Padding(8, 10).Gap(5).Children(func() {
				for _, e := range ents {
					e := e
					ui.Row(c).Gap(10).AlignItems(ui.Start).Children(func() {
						ui.Text(c, e.Key).Font("monospace").FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine().Width(kw).Shrink(0)
						ui.Column(c).Grow(1).MinWidth(0).Children(func() { a.fmValue(c, e) })
					})
				}
			})
		}
		if maxH > 0 {
			kit.VScroll(c).Key("mdfront-s-" + id).MaxHeight(maxH).Children(body)
		} else {
			body()
		}
	})
}

// fmValue 画一条属性的值。
func (a *App) fmValue(c *ui.Context, e fmEntry) {
	t := a.tk
	chip := func(s string, col ui.Color) {
		ui.Row(c).Padding(1, 6).Radius(t.RXs()).Background(t.Fill2).Shrink(0).Children(func() {
			ui.Text(c, s).Font("monospace").FontSize(theme.FsMicro).TextColor(col).SingleLine()
		})
	}
	switch e.Kind {
	case fmList:
		ui.Row(c).Gap(4).Wrap().Children(func() {
			for _, s := range e.List {
				chip(s, t.Tx2)
			}
		})
	case fmBool:
		col := t.Tx3
		if e.Text == "true" {
			col = t.Ok
		}
		ui.Row(c).Children(func() { chip(e.Text, col) })
	default:
		if strings.EqualFold(e.Key, "name") { // 名字是标识:加粗等宽
			ui.Text(c, e.Text).Font("monospace").FontSize(theme.FsCap).FontWeight(600).TextColor(t.Tx1)
			return
		}
		ui.Text(c, e.Text).FontSize(theme.FsCap).TextColor(t.Tx2).LineHeight(1.55)
	}
}

// fmKeyWidth 是键列宽:按最长的键估算(等宽小字约 6.4px 一个字符),夹在 64 ~ 168 之间,让各行的值左对齐。
func fmKeyWidth(ents []fmEntry) float32 {
	mx := 0
	for _, e := range ents {
		mx = max(mx, utf8.RuneCountInString(e.Key))
	}
	return theme.Px(min(max(float32(mx)*6.4+4, 64), 168))
}

// fmSummary 是折叠时标题行上的摘要。
func fmSummary(ents []fmEntry) string {
	for _, e := range ents {
		if strings.EqualFold(e.Key, "name") && e.Text != "" {
			return e.Text
		}
	}
	if len(ents) > 0 {
		e := ents[0]
		if e.Kind == fmList {
			return e.Key + ": " + strings.Join(e.List, ", ")
		}
		return e.Key + ": " + strings.ReplaceAll(e.Text, "\n", " ")
	}
	return ""
}

// fmStrip 去掉头首尾的 `---` 行(只用于解析失败时显示源码)。
func fmStrip(front string) string {
	lines := strings.Split(strings.TrimRight(strings.ReplaceAll(front, "\r\n", "\n"), "\n"), "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		lines = lines[1:]
	}
	if n := len(lines); n > 0 && strings.TrimSpace(lines[n-1]) == "---" {
		lines = lines[:n-1]
	}
	return strings.Join(lines, "\n")
}

// mdDoc 渲染一份可能带 YAML 头的 Markdown 全文:头是属性卡片,正文是 Markdown。
// id 区分折叠状态;maxH 见 mdFrontCard。头只认文件最开头的那一段,别处的 `---` 仍是分隔线。
func (a *App) mdDoc(c *ui.Context, id, src string, maxH float32) {
	src = strings.TrimPrefix(src, "\xef\xbb\xbf") // 去掉 UTF-8 BOM
	front, body := mdSplitFront(strings.ReplaceAll(src, "\r\n", "\n"))
	a.mdFrontCard(c, id, front, maxH)
	if strings.TrimSpace(body) == "" {
		return
	}
	markdown.View(c, padCode(body), a.mdOptions())
}
