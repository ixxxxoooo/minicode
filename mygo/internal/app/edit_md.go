package app

import (
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/egoist/mygo/ui/richtext"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// Markdown 所见即所得(设计稿 ide-md):默认以渲染后的文档打开并直接编辑,标签上的分段随时切回源码。
// 两种视图读写同一份缓冲区:所见即所得里每次修改都把文档序列化回缓冲区;缓冲区被别处改过(源码编辑、
// 格式化、磁盘重载)就从缓冲区重新读入文档。表格 / HTML / 脚注没法无损往返,这类文件只读预览,不冒险改写。

const (
	mdWysiwyg int8 = 1 // 标签上手动选了所见即所得
	mdSource  int8 = 2 // 标签上手动选了源码
)

// isMarkdownFile 按扩展名判断是不是 Markdown 文件。
func isMarkdownFile(p string) bool {
	switch strings.ToLower(path.Ext(p)) {
	case ".md", ".markdown", ".mdc":
		return true
	}
	return false
}

// mdState 是一个 Markdown 标签的所见即所得状态。
type mdState struct {
	doc    *richtext.Document
	ver    uint64 // 文档是按缓冲区的这一版建的
	front  string // 文件开头的 YAML 头(原样保留,不进文档)
	unsafe string // 非空 = 含无法往返的内容(表格 / HTML / 脚注),只读预览
	notes  int    // 解析再序列化后写法会变的行数
	edited bool   // 在所见即所得里改过
}

// mdSource 此刻该标签是否显示源码:标签上的选择优先,否则看偏好 editor.mdView。
func (a *App) mdSource(tb *ideTab) bool {
	switch tb.mdMode {
	case mdWysiwyg:
		return false
	case mdSource:
		return true
	}
	return a.prefs.Int(pMdView, 0) == 1
}

// setMdMode 手动切换视图(切到源码时顺便把光标交还编辑器)。
func (a *App) setMdMode(tb *ideTab, src bool) {
	if src {
		if tb.md != nil && tb.md.doc != nil {
			tb.md.doc.BreakUndoCoalescing()
		}
		tb.mdMode = mdSource
	} else {
		tb.mdMode = mdWysiwyg
		if tb.md != nil && tb.md.doc != nil {
			tb.md.doc.BreakUndoCoalescing()
		}
	}
	uxLog.Info("切换 Markdown 视图", "path", tb.path, "源码", src)
}

// toggleMdView 是命令面板 / 菜单里的「切换所见即所得 / 源码」。
func (a *App) toggleMdView() {
	if tb := a.ide.cur(); tb != nil && tb.buf != nil && isMarkdownFile(tb.path) {
		a.setMdMode(tb, !a.mdSource(tb))
	}
}

// mdSplitFront 把开头的 YAML 头(--- … ---,.mdc 规则文件常见)与正文分开。
func mdSplitFront(src string) (front, body string) {
	if !strings.HasPrefix(src, "---\n") {
		return "", src
	}
	rest := src[4:]
	i := strings.Index(rest, "\n---")
	if i < 0 {
		return "", src
	}
	end := 4 + i + 4
	if end < len(src) && src[end] != '\n' {
		return "", src
	}
	if end < len(src) {
		end++
	}
	return src[:end], src[end:]
}

var (
	mdTableSep = regexp.MustCompile(`^[\s|:\-]+$`)
	mdHTML     = regexp.MustCompile(`^\s*(<!--|</?[A-Za-z][A-Za-z0-9-]*(\s[^>]*)?/?>)`)
	mdFootnote = regexp.MustCompile(`^\[\^[^\]]+\]:`)
)

// mdUnsafe 找无法在所见即所得里无损表示的内容,返回它的名字("" = 没有)。围栏代码块里的不算。
func mdUnsafe(body string) string {
	fence := ""
	for _, ln := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(ln)
		if fence != "" {
			if strings.HasPrefix(trim, fence) {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(trim, "```") || strings.HasPrefix(trim, "~~~") {
			fence = trim[:3]
			continue
		}
		switch {
		case strings.Contains(trim, "|") && strings.Contains(trim, "-") && mdTableSep.MatchString(trim):
			return "表格"
		case mdHTML.MatchString(ln):
			return "HTML"
		case mdFootnote.MatchString(trim):
			return "脚注"
		}
	}
	return ""
}

// mdNormalized 粗略估计「解析再序列化」会改掉多少行:原文里找不到对应(去首尾空白后)的非空行数。
func mdNormalized(body string) int {
	have := map[string]int{}
	for _, l := range strings.Split(richtext.ParseMarkdown(body).Markdown(), "\n") {
		if s := strings.TrimSpace(l); s != "" {
			have[s]++
		}
	}
	n := 0
	for _, l := range strings.Split(body, "\n") {
		s := strings.TrimSpace(l)
		if s == "" {
			continue
		}
		if have[s] > 0 {
			have[s]--
		} else {
			n++
		}
	}
	return n
}

// mdEnsure 保证文档与缓冲区同步:缓冲区版本变了(源码里改过、格式化、磁盘重载)就重新读入。
func (a *App) mdEnsure(tb *ideTab) *mdState {
	md := tb.md
	if md == nil {
		md = &mdState{}
		tb.md = md
	}
	if md.doc != nil && md.ver == tb.buf.Version() {
		return md
	}
	defer diag.SpanSlow("edit", "Markdown 解析", 30*time.Millisecond, "path", tb.path)()
	src := tb.buf.String()
	front, body := mdSplitFront(src)
	md.front, md.edited = front, false
	md.unsafe = mdUnsafe(body)
	md.notes = 0
	if md.unsafe == "" {
		md.notes = mdNormalized(body)
		text := richtext.ParseMarkdown(body)
		if md.doc == nil {
			md.doc = richtext.NewDocument(text)
		} else {
			md.doc.SetText(text)
		}
		md.doc.SetSelection(0, 0) // 光标在开头:编辑器会滚到光标处,默认落在末尾会一上来就滚到文末
	}
	md.ver = tb.buf.Version()
	return md
}

// mdWrite 把文档序列化回缓冲区(所见即所得里的每次修改)。
func (a *App) mdWrite(tb *ideTab, md *mdState) {
	body := md.doc.Text().Markdown()
	out := md.front
	if body != "" {
		out += body + "\n"
	}
	orig := tb.buf.String()
	out = mdPreserveRoundTrip(orig, out)
	if out == orig {
		return
	}
	tb.buf.Set(out)
	md.ver = tb.buf.Version()
	md.edited = true
}

// rtTheme 把当前令牌映射成富文本编辑器的外观(随主题缓存)。
func (a *App) rtTheme() *richtext.Theme {
	if a.ux.rt != nil {
		return a.ux.rt
	}
	t := a.tk
	th := richtext.DefaultTheme(t.UITheme())
	th.Text, th.Muted, th.Link = t.Tx1, t.Tx4, t.AcTx
	th.Caret, th.Selection = t.Ac, t.Ac.Alpha(.22)
	th.Code, th.CodeBackground = t.Tx1, t.Fill2
	th.CodeBlockBackground = t.Sunken
	th.Marker, th.Checkbox = t.Tx3, t.Ac
	th.QuoteBar, th.QuoteText = t.Bd3, t.Tx2
	for i := range th.Headings {
		th.Headings[i].Color = t.Tx1
	}
	sizes := [3]float32{1.75, 1.4, 1.18}
	for i, sz := range sizes {
		th.Headings[i].Size, th.Headings[i].Weight = sz, 700
		th.Headings[i].Top, th.Headings[i].Bottom = .9, .35
	}
	th.ParagraphSpacing = .7
	th.CodeBlockRadius = float32(t.RMd()) / theme.FsRead
	a.ux.rt = th
	return th
}

// mdView 是 Markdown 标签的所见即所得内容(顶部已有面包屑条)。
func (a *App) mdView(c *ui.Context, tb *ideTab) {
	t := a.tk
	md := a.mdEnsure(tb)
	ui.Column(c).Grow(1).MinHeight(0).Children(func() {
		switch {
		case md.unsafe != "":
			a.mdBanner(c, tb, true, "这个文件含"+md.unsafe+",所见即所得会把它压成普通段落,所以只读显示。", nil)
			ui.Scroll(c).Key("mdprev-" + tb.id).Grow(1).MinHeight(0).Children(func() {
				ui.Column(c).Padding(14, 34, 28, 34).MaxWidth(860).Children(func() {
					markdown.View(c, tb.buf.String(), a.mdOptions())
				})
			})
			return
		case md.notes > 0 && !md.edited && !a.ux.mdNoteOff[tb.id]:
			a.mdBanner(c, tb, false, "有 "+itoa(md.notes)+" 行的写法(如代码块语言标注、段内换行)在所见即所得里会被规范化。只有你开始编辑后才会改写文件。", func() {
				kit.Btn(c, "知道了", kit.BtnOpt{Variant: kit.BtnGhost, Small: true}).OnClick(func() {
					if a.ux.mdNoteOff == nil {
						a.ux.mdNoteOff = map[string]bool{}
					}
					a.ux.mdNoteOff[tb.id] = true
				})
			})
		}
		a.mdFront(c, md.front)
		a.mdFormatBar(c, md.doc)
		a.mdApplyReveal(tb, md)
		ui.Row(c).Grow(1).MinHeight(0).Justify(ui.Center).AlignItems(ui.Stretch).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).MaxWidth(900).Padding(10, 30, 6, 30).Children(func() {
				ed := richtext.EditorBase(c.Key("md-"+tb.id), md.doc, richtext.EditorOptions{
					Label:       "Markdown 文档",
					Placeholder: "开始写点什么……  # 空格 变标题,- 空格 变列表",
					Font:        ui.Font{Size: 14},
					Theme:       a.rtTheme(),
					OnLink:      a.openMdLink,
					OnKey: func(key ui.Key, mods ui.Modifiers) bool {
						if mods == ui.Cmd && key == ui.KeyV {
							if a.mdTryPasteImage(c, tb, md) {
								return true
							}
						}
						return false
					},
				}).Fill()
				if ed.Changed() {
					a.mdWrite(tb, md)
				}
			})
		})
	})
	_ = t
}

// mdBanner 是视图顶部的一条提示:警告(只读预览)或信息(写法会被规范化),带「用源码编辑」。
func (a *App) mdBanner(c *ui.Context, tb *ideTab, warn bool, text string, extra func()) {
	t := a.tk
	bg, icon, col := t.Fill1, "info", t.Info
	if warn {
		bg, icon, col = t.WarnSoft, "alert", t.Warn
	}
	ui.Row(c).Key("mdbanner").Padding(6, 12).Gap(8).Shrink(0).AlignItems(ui.Center).Background(bg).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
		kit.Ic(c, icon, kit.S12).TextColor(col)
		ui.Text(c, text).Grow(1).MinWidth(0).FontSize(theme.FsCap).TextColor(t.Tx2).LineHeight(1.5)
		kit.Btn(c, "用源码编辑", kit.BtnOpt{Small: true, Icon: "code"}).OnClick(func() { a.setMdMode(tb, true) })
		if extra != nil {
			extra()
		}
	})
}

// mdFormatBar 是所见即所得的格式条:行内格式 | 标题 | 列表 / 引用 / 代码块;按钮高亮取自光标所在处的格式。
func (a *App) mdFormatBar(c *ui.Context, doc *richtext.Document) {
	t := a.tk
	f := doc.Formatting()
	ui.Row(c).Height(32).Shrink(0).PaddingX(10).Gap(1).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
		btn := func(key, tip string, on bool, content func(), run func()) {
			b := ui.ButtonBase(c).Key("mdf-" + key).MinWidth(24).Height(24).PaddingX(5).Radius(t.RSm()).Justify(ui.Center).AlignItems(ui.Center).
				Cursor(ui.CursorPointer).TextColor(t.Tx2).FontSize(12).Tooltip(tip)
			switch {
			case on:
				b.Background(t.Fill2).TextColor(t.Tx1)
			case b.Hovered():
				b.Background(t.Fill1).TextColor(t.Tx1)
			}
			if b.Clicked() {
				run()
			}
			b.Children(content)
		}
		text := func(s string, w int, italic bool) func() {
			return func() {
				x := ui.Text(c, s).FontWeight(w)
				if italic {
					x.Italic()
				}
			}
		}
		icon := func(n string) func() { return func() { kit.Ic(c, n, kit.S12) } }
		sep := func() { ui.Box(c).Size(1, 14).Margin(0, 4).Background(t.Bd2) }
		on := func(s richtext.FormatState) bool { return s == richtext.FormatOn }
		btn("b", "粗体 ⌘B", on(f.Bold), text("B", 800, false), doc.ToggleBold)
		btn("i", "斜体 ⌘I", on(f.Italic), text("I", 500, true), doc.ToggleItalic)
		btn("s", "删除线 ⇧⌘X", on(f.Strikethrough), text("S", 500, false), doc.ToggleStrikethrough)
		btn("c", "行内代码 ⌘E", on(f.Code), icon("code"), doc.ToggleCode)
		btn("l", "链接:选中文字后点,剪贴板里的网址会作为链接目标 ⌘K", f.Link != "", icon("link"), func() {
			if f.Link != "" {
				doc.SetLink("")
				return
			}
			if u := strings.TrimSpace(c.ReadClipboard()); strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
				doc.SetLink(u)
				return
			}
			a.toast = "先复制一个网址,再选中文字点链接(或选中网址按 ⌘K)"
		})
		sep()
		h := f.Paragraph.Heading
		if f.ParagraphsMixed {
			h = -1
		}
		for n := 1; n <= 3; n++ {
			lv := n
			btn("h"+itoa(lv), "标题 "+itoa(lv), h == lv, text("H"+itoa(lv), 600, false), func() {
				if h == lv {
					doc.SetHeading(0)
				} else {
					doc.SetHeading(lv)
				}
			})
		}
		sep()
		lk := f.Paragraph.List
		if f.ParagraphsMixed {
			lk = richtext.NoList
		}
		btn("ul", "无序列表", lk == richtext.Bullets, icon("list"), func() { doc.ToggleList(richtext.Bullets) })
		btn("ol", "有序列表", lk == richtext.Numbers, text("1.", 500, false), func() { doc.ToggleList(richtext.Numbers) })
		btn("td", "任务列表", lk == richtext.Tasks, icon("todo"), func() { doc.ToggleList(richtext.Tasks) })
		btn("q", "引用", f.Paragraph.Quote && !f.ParagraphsMixed, icon("quote"), doc.ToggleQuote)
		btn("cb", "代码块", f.Paragraph.CodeBlock && !f.ParagraphsMixed, text("{ }", 500, false), doc.ToggleCodeBlock)
		ui.Spacer(c).Grow(1)
		ui.Text(c, "# 空格 · - 空格 · ``` 即时生效").FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine().Shrink(1)
	})
}

// langName 是面包屑条右侧的语言名(按扩展名,认不出就不显示)。
func langName(p string) string {
	switch strings.ToLower(path.Ext(p)) {
	case ".md", ".markdown", ".mdc":
		return "Markdown"
	case ".rs":
		return "Rust"
	case ".go":
		return "Go"
	case ".ts", ".tsx":
		return "TypeScript"
	case ".js", ".jsx", ".mjs":
		return "JavaScript"
	case ".py":
		return "Python"
	case ".json":
		return "JSON"
	case ".toml":
		return "TOML"
	case ".yaml", ".yml":
		return "YAML"
	case ".sh", ".zsh":
		return "Shell"
	case ".html":
		return "HTML"
	case ".css":
		return "CSS"
	case ".txt":
		return "纯文本"
	}
	return ""
}

// mdFront 把文件开头的 YAML 头显示成一条只读的「属性」条(.mdc 规则文件常见);要改它请切到源码。
func (a *App) mdFront(c *ui.Context, front string) {
	if front == "" {
		return
	}
	t := a.tk
	ui.Row(c).Key("mdfront").Shrink(0).Padding(5, 12).Gap(8).AlignItems(ui.Center).Background(t.Fill1).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
		ui.Text(c, "属性").FontSize(theme.FsMicro).FontWeight(600).TextColor(t.Tx4).Shrink(0)
		for _, ln := range strings.Split(front, "\n") {
			k, v, ok := strings.Cut(ln, ":")
			if !ok || strings.TrimSpace(k) == "" || strings.HasPrefix(ln, " ") {
				continue
			}
			ui.Row(c).Gap(4).Padding(1, 6).Radius(t.RXs()).Background(t.Fill2).Shrink(1).MinWidth(0).Children(func() {
				ui.Text(c, strings.TrimSpace(k)).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine().Shrink(0)
				ui.Text(c, strings.TrimSpace(v)).FontSize(theme.FsCap).Font("monospace").SingleLine().Shrink(1).MinWidth(0)
			})
		}
	})
}
