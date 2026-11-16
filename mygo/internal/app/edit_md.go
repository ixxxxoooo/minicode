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
	ver    uint64      // 文档是按缓冲区的这一版建的
	front  string      // 文件开头的 YAML 头(原样保留,不进文档)
	unsafe string      // 非空 = 含无法往返的内容(HTML / 脚注 / 列表里的表格),只读预览
	notes  int         // 若编辑对应块会被规范化的非空行数
	edited bool        // 在所见即所得里改过
	body   mdBodySplit // 读入时切分的正文块(写回时只替换变了的块)

	loaded   bool     // 已按 ver 这一版读入过(只读预览也没有 doc,不能再靠 doc 判断)
	segs     []*mdSeg // 含顶层表格时:按「正文 / 表格」切成的段(此时 doc 为 nil),见 edit_md_table.go
	pend     []mdPend // 表格上排队等执行的结构修改
	tblFocus string   // 下一次构建里要聚焦的表格单元格键
	readOnly bool     // 用户手动开启的只读预览(与 mdUnsafe 自动只读共用渲染路径)
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

// mdPreviewReadOnly 预览是否不可编辑(含自动只读与手动开关;源码视图不算预览)。
func (a *App) mdPreviewReadOnly(tb *ideTab) bool {
	if tb == nil || tb.md == nil || a.mdSource(tb) {
		return false
	}
	return tb.md.unsafe != "" || tb.md.readOnly
}

// toggleMdReadOnly 切换手动只读预览(自动只读时应从菜单置灰,不走这里)。
func (a *App) toggleMdReadOnly(tb *ideTab) {
	if tb == nil || tb.buf == nil || !isMarkdownFile(tb.path) {
		return
	}
	md := a.mdEnsure(tb)
	md.readOnly = !md.readOnly
	uxLog.Info("Markdown 只读预览", "path", tb.path, "on", md.readOnly)
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
// 顶层表格有自己的可编辑网格(edit_md_table.go),不算;缩进在列表 / 引用里的表格仍算。
func mdUnsafe(body string) string {
	for _, s := range mdSplitSegs(body) {
		if s.table {
			continue
		}
		if u := mdUnsafeText(s.pre + s.raw + s.post); u != "" {
			return u
		}
	}
	return ""
}

func mdUnsafeText(body string) string {
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

// mdNormalized 统计编辑时可能被规范化的非空行数(只计会变的块)。
func mdNormalized(body string) int {
	return mdNormalizedBlocks(body)
}

// mdEnsure 保证文档与缓冲区同步:缓冲区版本变了(源码里改过、格式化、磁盘重载)就重新读入。
func (a *App) mdEnsure(tb *ideTab) *mdState {
	md := tb.md
	if md == nil {
		md = &mdState{}
		tb.md = md
	}
	if md.loaded && md.ver == tb.buf.Version() {
		return md
	}
	defer diag.SpanSlow("edit", "Markdown 解析", 30*time.Millisecond, "path", tb.path)()
	src := tb.buf.String()
	front, body := mdSplitFront(src)
	md.front, md.edited = front, false
	md.unsafe = mdUnsafe(body)
	md.notes = 0
	md.body = mdSplitBodyBlocks(body)
	md.segs, md.pend = nil, nil
	if md.unsafe != "" {
		md.doc = nil
	} else if segs := mdSplitSegs(body); mdSegsHaveTable(segs) {
		// 含顶层表格:按段编辑,正文段各一份富文本文档,表格段是网格
		md.doc = nil
		for _, sg := range segs {
			if sg.table {
				continue
			}
			md.notes += mdNormalized(sg.raw)
			sg.doc = richtext.NewDocument(richtext.ParseMarkdown(sg.raw))
			sg.doc.SetSelection(0, 0)
			sg.h = float32(sg.doc.Text().ParagraphCount())*30 + 30
		}
		md.segs = segs
	} else {
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
	md.loaded = true
	return md
}

// mdWrite 把文档序列化回缓冲区(所见即所得里的每次修改);只替换被编辑过的块。
func (a *App) mdWrite(tb *ideTab, md *mdState) {
	serialized := md.doc.Text().Markdown()
	_, origBody := mdSplitFront(tb.buf.String())
	merged := mdPreserveBody(origBody, md.body, serialized)
	out := md.front
	if merged != "" {
		out += merged
		if !strings.HasSuffix(out, "\n") {
			out += "\n"
		}
	}
	orig := tb.buf.String()
	if out == orig {
		return
	}
	tb.buf.Set(out)
	md.ver = tb.buf.Version()
	md.edited = true
	_, body := mdSplitFront(out)
	md.body = mdSplitBodyBlocks(body)
	uxLog.Debug("Markdown 写回", "path", tb.path, "blocks", len(md.body.parts))
}

// rtTheme 把当前令牌映射成富文本编辑器的外观(随主题缓存),对照设计稿 ide-md 的 .md 样式:
// 正文 13px / 行高 1.7,H1 21 · H2 15 · H3 14,行内代码取强调色(字 ac-tx、底 ac 11%),
// 列表符 tx-4,引用左侧 3px 条 + tx-2 字,任务框用成功色。
func (a *App) rtTheme() *richtext.Theme {
	if a.ux.rt != nil {
		return a.ux.rt
	}
	t := a.tk
	th := richtext.DefaultTheme(t.UITheme())
	th.Text, th.Muted, th.Link = t.Tx1, t.Tx3, t.AcTx
	th.Caret, th.Selection = t.Ac, t.Ac.Alpha(.22)
	th.Code, th.CodeBackground = a.inlineCodeColors()
	th.CodeSize = 11.5 / theme.FsRead
	th.CodeBlockBackground = t.Sunken
	th.CodeBlockPadding = [2]float32{.8, .55}
	th.Marker, th.Checkbox = t.Tx4, t.Ok
	th.QuoteBar, th.QuoteText = t.Bd3, t.Tx2
	th.QuoteBarWidth = 3
	th.Indent = 1.4
	for i := range th.Headings {
		th.Headings[i].Color = t.Tx1
	}
	// 设计稿:h1 21 / 700(下 8)、h2 15 / 650(上 12 下 4)、h3 14 / 650(上 8 下 4);Size 以正文 13px 为 1。
	heads := [3]richtext.HeadingStyle{
		{Size: 21.0 / theme.FsRead, Weight: 700, Color: t.Tx1, Top: .6, Bottom: .6},
		{Size: 15.0 / theme.FsRead, Weight: 650, Color: t.Tx1, Top: .8, Bottom: .3},
		{Size: 14.0 / theme.FsRead, Weight: 650, Color: t.Tx1, Top: .55, Bottom: .3},
	}
	copy(th.Headings[:3], heads[:])
	th.ParagraphSpacing = .5
	th.CodeBlockRadius = float32(t.RMd()) / theme.FsRead
	a.ux.rt = th
	return th
}

// rtFont 是预览正文的字体:设计稿 .md 的 13px / 行高 1.7。
func rtFont() ui.Font { return ui.Font{Size: theme.FsRead, LineHeight: 1.7} }

// rtSpans 微调预览里个别段落的着色:代码块恢复正文色(行内代码取强调色,不能殃及代码块),
// 已完成任务不加删除线、不变灰(设计稿里只靠实心勾框区分)。
func (a *App) rtSpans(p int, st richtext.ParagraphStyle, spans []ui.Span) []ui.Span {
	t := a.tk
	switch {
	case st.CodeBlock:
		for i := range spans {
			spans[i].Color = t.Tx1
		}
	case st.Checked:
		for i := range spans {
			spans[i].Strikethrough = false
			spans[i].Color = t.Tx2
		}
	}
	return spans
}

// mdView 是 Markdown 标签的所见即所得内容(顶部已有面包屑条)。
func (a *App) mdView(c *ui.Context, tb *ideTab) {
	t := a.tk
	md := a.mdEnsure(tb)
	ui.Column(c).Grow(1).MinHeight(0).Children(func() {
		switch {
		case a.mdPreviewReadOnly(tb):
			if md.unsafe != "" {
				a.mdBanner(c, tb, true, "这个文件含"+md.unsafe+",预览里编辑会把它压成普通段落,所以只读显示。", nil)
			}
			kit.VScroll(c).Key("mdprev-" + tb.id).Grow(1).MinHeight(0).Children(func() {
				ui.Column(c).Padding(14, 34, 28, 34).MaxWidth(860).Children(func() {
					markdown.View(c, tb.buf.String(), a.mdOptions())
				})
			})
			return
		case md.notes > 0 && !md.edited && !a.ux.mdNoteOff[tb.id]:
			a.mdBanner(c, tb, false, "有 "+itoa(md.notes)+" 行写法与预览序列化不完全一致(如段内换行、列表符、围栏语言)。编辑对应段落后才会写回,未改动的段落保持原样。", func() {
				kit.Btn(c, "知道了", kit.BtnOpt{Variant: kit.BtnGhost, Small: true}).OnClick(func() {
					if a.ux.mdNoteOff == nil {
						a.ux.mdNoteOff = map[string]bool{}
					}
					a.ux.mdNoteOff[tb.id] = true
				})
			})
		}
		a.mdFront(c, md.front)
		if md.segs != nil { // 含表格:正文 + 可编辑网格分段显示
			a.mdApplyReveal(tb, md)
			a.mdSegsView(c, tb, md)
			return
		}
		if mdShowFormatBar { // 预览不再显示格式条(设计稿 ide-md),快捷键照常生效
			a.mdFormatBar(c, md.doc)
		}
		a.mdApplyReveal(tb, md)
		ui.Row(c).Grow(1).MinHeight(0).Justify(ui.Center).AlignItems(ui.Stretch).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).MaxWidth(900).Padding(14, 34, 8, 34).Children(func() {
				// 库里 Box.Clip 只裁子元素、不裁自己的 Draw,富文本编辑器是自绘的,滚动后整段会画出自己的矩形
				// (盖住上面的提示条、下面的底部面板),所以套一层不带内边距的裁剪容器。
				ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).Clip().Children(func() {
					ed := richtext.EditorBase(c.Key("md-"+tb.id), md.doc, richtext.EditorOptions{
						Label:        "Markdown 文档",
						Placeholder:  "开始写点什么……  # 空格 变标题,- 空格 变列表",
						Font:         rtFont(),
						Theme:        a.rtTheme(),
						Spans:        a.rtSpans,
						SpansVersion: 1,
						OnLink:       a.openMdLink,
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

// mdShowFormatBar 控制预览顶部的格式条是否显示(默认隐藏,函数保留以便日后恢复)。
const mdShowFormatBar = false

// mdFormatBar 是所见即所得的格式条:行内格式 | 标题 | 列表 / 引用 / 代码块;按钮高亮取自光标所在处的格式。
func (a *App) mdFormatBar(c *ui.Context, doc *richtext.Document) {
	t := a.tk
	f := doc.Formatting()
	ui.Row(c).Height(theme.Px(32)).Shrink(0).PaddingX(10).Gap(1).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
		btn := func(key, tip string, on bool, content func(), run func()) {
			b := ui.ButtonBase(c).Key("mdf-" + key).MinWidth(24).Height(theme.Px(24)).PaddingX(5).Radius(t.RSm()).Justify(ui.Center).AlignItems(ui.Center).
				Cursor(ui.CursorPointer).TextColor(t.Tx2).FontSize(theme.Px(12)).Tooltip(tip)
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
		ui.Text(c, "属性").FontSize(theme.FsMicro).FontWeight(500).TextColor(t.Tx4).Shrink(0)
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
