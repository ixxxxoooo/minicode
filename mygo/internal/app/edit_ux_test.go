package app

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
)

func TestRefTextAndAppend(t *testing.T) {
	if refText("src/a.rs", false) != "@src/a.rs" || refText("docs", true) != "@docs/" || refText("docs/", true) != "@docs/" {
		t.Fatal("引用文字不对")
	}
	cases := []struct {
		draft string
		refs  []string
		want  string
	}{
		{"", []string{"@a"}, "@a "},
		{"看一下", []string{"@a"}, "看一下 @a "},
		{"看一下 ", []string{"@a", "@b"}, "看一下 @a @b "},
		{"看 @a 吧", []string{"@a"}, "看 @a 吧"},    // 已引用不重复
		{"x\n", []string{"@a", "@a"}, "x\n@a "}, // 本次重复也只写一次
		{"@ab", []string{"@a"}, "@ab @a "},      // 前缀相同不算已引用
	}
	for _, c := range cases {
		if got := appendRefs(c.draft, c.refs...); got != c.want {
			t.Errorf("appendRefs(%q,%v)=%q,想要 %q", c.draft, c.refs, got, c.want)
		}
	}
}

func TestBuiltinFormat(t *testing.T) {
	// Go:gofmt
	out, by, err := builtinFormat("a.go", "package a\nfunc  f( ){\nx:=1\n_=x}\n", 2)
	if err != nil || by != "Go" || !strings.Contains(out, "func f() {") {
		t.Fatalf("Go 格式化不对:%q %q %v", out, by, err)
	}
	if _, _, err := builtinFormat("a.go", "package a\nfunc (", 2); err == nil {
		t.Fatal("语法错误的 Go 应报错")
	}
	// JSON:按缩进宽度整理,非法的报错且原样返回
	out, _, err = builtinFormat("a.json", `{"a":[1,2],"b":{"c":1}}`, 2)
	if err != nil || !strings.Contains(out, "{\n  \"a\": [") || !strings.HasSuffix(out, "}\n") {
		t.Fatalf("JSON 格式化不对:%q %v", out, err)
	}
	out, _, _ = builtinFormat("a.json", `{"a":1}`, 4)
	if !strings.Contains(out, "\n    \"a\"") {
		t.Fatalf("4 空格缩进不对:%q", out)
	}
	if o, _, err := builtinFormat("a.json", `{"a":`, 2); err == nil || o != `{"a":` {
		t.Fatal("非法 JSON 应报错并原样返回")
	}
	// 文本:去行尾空白,末尾恰好一个换行,保留 CRLF
	if got := formatText("a  \nb\t\n\n\n"); got != "a\nb\n" {
		t.Fatalf("文本整理不对:%q", got)
	}
	if got := formatText("a \r\nb\r\n\r\n"); got != "a\r\nb\r\n" {
		t.Fatalf("CRLF 应保留:%q", got)
	}
	if formatText("") != "" {
		t.Fatal("空文件保持为空")
	}
	// Markdown:围栏里不动,硬换行保留,空行压缩
	src := "\n\n# 标题  \n\n\n\n正文  \n下一行 \n\n```\n  代码   \n\n\n\n```\n"
	want := "# 标题\n\n正文  \n下一行\n\n```\n  代码   \n\n\n\n```\n"
	if got := formatMarkdown(src); got != want {
		t.Fatalf("Markdown 整理不对:\n%q\n想要\n%q", got, want)
	}
	// 幂等
	if got := formatMarkdown(want); got != want {
		t.Fatalf("格式化应当幂等:%q", got)
	}
}

func TestMarkdownHelpers(t *testing.T) {
	f, b := mdSplitFront("---\ndescription: x\n---\n\n# 标题\n")
	if f != "---\ndescription: x\n---\n" || b != "\n# 标题\n" {
		t.Fatalf("YAML 头拆分不对:%q | %q", f, b)
	}
	if f, b := mdSplitFront("# 没有头\n"); f != "" || b != "# 没有头\n" {
		t.Fatal("没有 YAML 头时原样返回")
	}
	if f, _ := mdSplitFront("---\n没有结束\n"); f != "" {
		t.Fatal("没有结束线不算 YAML 头")
	}
	for src, want := range map[string]string{
		"| a | b |\n| --- | --- |\n| 1 | 2 |\n": "",   // 顶层表格有可编辑网格,不算
		"- 项\n  | a |\n  | - |\n":               "表格", // 缩进在列表里的表格写回会丢缩进,仍算
		"正文\n\n<details>\n</details>\n":         "HTML",
		"见[^1]\n\n[^1]: 脚注\n":                   "脚注",
		"```\n| a |\n| - |\n<div>\n```\n":       "", // 代码块里的不算
		"---\n\n正文 <https://x.dev>\n":           "", // 分隔线与自动链接不算
		"# 普通\n\n- 列表\n":                        "",
	} {
		if got := mdUnsafe(src); got != want {
			t.Errorf("mdUnsafe(%q)=%q,想要 %q", src, got, want)
		}
	}
}

// 所见即所得:默认打开、文档与缓冲区双向同步、表格只读预览、偏好与标签选择。
func TestMarkdownWysiwyg(t *testing.T) {
	a, tt := demoIDE(t)
	d := a.ide
	d.openFile("docs/guide.md", true)
	tt.Frame()
	tb := d.cur()
	if tb == nil || tb.md == nil || tb.md.doc == nil || tb.md.unsafe != "" {
		t.Fatalf("Markdown 应默认以所见即所得打开:%+v", tb.md)
	}
	if a.mdSource(tb) {
		t.Fatal("默认不是源码视图")
	}
	// 文档里改了 → 序列化回缓冲区,文件变脏
	md := tb.md
	md.doc.SetText(richtext.ParseMarkdown("# 改过\n\n新内容\n"))
	a.mdWrite(tb, md)
	if tb.buf.String() != "# 改过\n\n新内容\n" || !tb.dirty() || !md.edited {
		t.Fatalf("写回缓冲区不对:%q", tb.buf.String())
	}
	tt.Frame() // 缓冲区版本与文档一致:不应重新读入
	if md.ver != tb.buf.Version() {
		t.Fatal("写回后版本应对齐")
	}
	// 源码里改了 → 下一帧文档重新读入
	tb.buf.Set("# 源码里改的\n")
	tt.Frame()
	if got := md.doc.Text().Markdown(); got != "# 源码里改的" {
		t.Fatalf("缓冲区变化后文档应重新读入:%q", got)
	}
	// 标签上手动选源码优先于偏好;偏好选源码则默认源码
	a.setMdMode(tb, true)
	if !a.mdSource(tb) {
		t.Fatal("手动切到源码")
	}
	a.toggleMdView()
	if a.mdSource(tb) {
		t.Fatal("切换后应回到所见即所得")
	}
	tb.mdMode = 0
	a.prefs.Set(pMdView, 1)
	if !a.mdSource(tb) {
		t.Fatal("偏好为源码时默认源码")
	}
	a.prefs.Reset(pMdView)

	// 含顶层表格:按段预览(正文 + 可编辑网格),不建整篇文档,打开也不改写文件
	d.openFile("docs/api.md", true)
	tt.Frame()
	api := d.cur()
	if api.md == nil || api.md.unsafe != "" || api.md.doc != nil || !mdSegsHaveTable(api.md.segs) {
		t.Fatalf("含表格应分段预览:%+v", api.md)
	}
	if api.dirty() {
		t.Fatal("只浏览不能弄脏文件")
	}

	// 含 HTML:只读预览,不建文档、不改写文件
	d.openFile("docs/embed.md", true)
	tt.Frame()
	emb := d.cur()
	if emb.md == nil || emb.md.unsafe != "HTML" || emb.md.doc != nil || emb.md.segs != nil {
		t.Fatalf("含 HTML 应只读预览:%+v", emb.md)
	}
	if emb.dirty() {
		t.Fatal("只读预览不能弄脏文件")
	}

	// 带 YAML 头的规则文件:头原样保留
	d.openFile(".cursor/rules/style.mdc", true)
	tt.Frame()
	rule := d.cur()
	if rule.md == nil || rule.md.doc == nil || !strings.HasPrefix(rule.md.front, "---\ndescription: 代码风格") {
		t.Fatalf("YAML 头应单独保留:%+v", rule.md)
	}
	before := rule.buf.String()
	rule.md.doc.SetText(richtext.ParseMarkdown("# 风格\n\n- 只改了这里\n"))
	a.mdWrite(rule, rule.md)
	got := rule.buf.String()
	if !strings.HasPrefix(got, rule.md.front) || !strings.HasSuffix(got, "- 只改了这里\n") || got == before {
		t.Fatalf("写回应保留 YAML 头:%q", got)
	}
}

func TestFormatBuiltinOnTab(t *testing.T) {
	a, tt := demoIDE(t)
	a.ide.openFile("docs/routing.md", true)
	tt.Frame()
	tb := a.ide.cur()
	tb.buf.Set("# 标题   \n\n\n\n正文\n")
	a.formatDoc() // 没有语言服务器 → 内置
	if tb.buf.String() != "# 标题\n\n正文\n" || !strings.Contains(a.toast, "已格式化") {
		t.Fatalf("内置格式化没生效:%q toast=%q", tb.buf.String(), a.toast)
	}
	a.formatDoc()
	if a.toast != "已经是格式化好的" {
		t.Fatalf("已整齐应提示:%q", a.toast)
	}
	// 保存时去行尾空白
	a.prefs.Set(pTrimSave, true)
	a.ide.openFile("README.md", true)
	tt.Frame()
	rd := a.ide.cur()
	a.setMdMode(rd, true) // 源码模式才整理
	rd.buf.Set("# r   \n\n\n正文  \n")
	if err := a.ide.save(); err != nil {
		t.Skip("演示文件系统不可写:", err)
	}
	if rd.buf.String() != "# r\n\n\n正文\n" {
		t.Fatalf("保存时应去掉行尾空白:%q", rd.buf.String())
	}
	a.prefs.Reset(pTrimSave)
}

func TestWrapToggleAndShortcut(t *testing.T) {
	a, tt := demoIDE(t)
	a.ide.openFile("src/router.rs", true)
	tt.Frame()
	if a.prefs.Bool(pEdWrap, false) {
		t.Fatal("默认不换行")
	}
	tt.ClickAt(640, 320) // 聚焦编辑器
	tt.Frame()
	tt.Key(ui.Alt, ui.KeyZ)
	tt.Frame()
	if !a.prefs.Bool(pEdWrap, false) || !a.ide.cur().st.SoftWrap {
		t.Fatal("⌥Z 应开启自动换行并作用到编辑器")
	}
	a.toggleWrap()
	if a.prefs.Bool(pEdWrap, false) {
		t.Fatal("再切一次应关闭")
	}
}

func TestTabMenusActions(t *testing.T) {
	a, tt := demoIDE(t)
	d := a.ide
	d.tabs, d.active = nil, ""
	for _, p := range []string{"README.md", "docs/guide.md", "src/router.rs", "tests/case_test.rs"} {
		d.openFile(p, true)
	}
	tt.Frame()
	d.tab("docs/guide.md").buf.Set("# 改了\n") // 未保存
	// 关闭已保存:只留下脏标签
	n := a.closeTabsWhere(func(tb *ideTab) bool { return tb.dirty() })
	if n != 3 || len(d.tabs) != 1 || d.tabs[0].path != "docs/guide.md" {
		t.Fatalf("关闭已保存应只留脏标签:n=%d tabs=%d", n, len(d.tabs))
	}
	// 已关闭的记在栈里,⇧⌘T 逆序重开
	if len(d.closed) != 3 {
		t.Fatalf("应记下 3 个已关闭标签:%v", d.closed)
	}
	last := d.closed[len(d.closed)-1]
	a.reopenClosed()
	if d.tab(last) == nil || len(d.closed) != 2 {
		t.Fatalf("应重新打开 %s:closed=%v", last, d.closed)
	}
	// 关闭右侧 / 其他
	d.openFile("README.md", true)
	ix := a.tabIndex("docs/guide.md")
	a.closeTabsWhere(func(x *ideTab) bool { return a.tabIndex(x.id) <= ix })
	if a.tabIndex("docs/guide.md") != len(d.tabs)-1 {
		t.Fatal("关闭右侧后当前标签应是最后一个")
	}
	a.closeTabsWhere(func(x *ideTab) bool { return x.id == "docs/guide.md" })
	if len(d.tabs) != 1 {
		t.Fatalf("关闭其他后只剩一个:%d", len(d.tabs))
	}
	// 在文件树中定位:展开各级目录并选中
	a.nav, d.sel = "sessions", ""
	a.revealInTree("src/handlers/users.rs")
	if a.nav != "files" || !d.open["src"] || !d.open["src/handlers"] || d.sel != "src/handlers/users.rs" {
		t.Fatalf("定位文件树不对:nav=%s open=%v sel=%s", a.nav, d.open, d.sel)
	}
}

// 拖拽:文件树里的文件拖到对话栏 = 在输入框里插入 @ 引用;访达拖入的文件同理;落在别处什么都不发生。
func TestDragFileToChat(t *testing.T) {
	a, tt := demoIDE(t)
	a.rightOn = true
	a.nav, a.sideOpen = "files", true
	tt.Frame()
	tt.Advance(time.Second) // 等形态切换的宽度过渡走完
	tt.Frame()
	chatX, chatY := float32(1280-150), float32(400) // IDE 形态:对话栏停靠在右侧
	if r, ok := tt.Find("描述任务"); ok {
		chatX, chatY = r.X+r.W/2, r.Y+r.H/2
	}
	// 访达拖入
	a.draft, a.refs = "", nil
	if !tt.DropFiles(chatX, chatY, "/x/y.txt") {
		t.Fatal("对话栏应接住拖入的文件")
	}
	if len(a.refs) != 1 || a.refs[0] != "@/x/y.txt" {
		t.Fatalf("应以胶囊引用绝对路径:%v", a.refs)
	}
	if a.draft != "" {
		t.Fatalf("引用胶囊不应再往输入框正文里写 @ 文字:%q", a.draft)
	}
	tt.Frame()
	if a.ux.focusDraft {
		t.Fatal("焦点交给输入框后标志应清掉")
	}
	// 落在编辑区:不引用
	a.draft, a.refs = "", nil
	tt.DropFiles(500, 300, "/x/z.txt")
	if a.draft != "" || len(a.refs) != 0 {
		t.Fatalf("落在对话栏以外不应引用:%q %v", a.draft, a.refs)
	}
	// 文件树 → 对话栏(无头测试里用右键「引用到对话」,与拖放共用 dropRefs)
	if err := tt.RightClick("router.rs"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("引用到对话"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(a.refs) != 1 || !strings.Contains(a.refs[0], "router.rs") {
		t.Fatalf("文件树引用到对话栏应加一枚引用胶囊,得 %v", a.refs)
	}
}

// 引用胶囊:去重、可移除、发送时并进正文最前并清空;正文里已写过的也算重复。
func TestRefChipsFold(t *testing.T) {
	a := &App{}
	if n := a.addRefs("@a.go", "@b/", "@a.go"); n != 2 || len(a.refs) != 2 {
		t.Fatalf("应去重:n=%d refs=%v", n, a.refs)
	}
	a.draft = "看 @c.go 吧"
	if a.addRefs("@c.go") != 0 {
		t.Fatal("正文里已有的引用不应再加胶囊")
	}
	a.removeRef("@b/")
	if len(a.refs) != 1 || a.refs[0] != "@a.go" {
		t.Fatalf("移除不对:%v", a.refs)
	}
	if got := a.foldRefs("解释一下"); got != "@a.go 解释一下" || a.refs != nil {
		t.Fatalf("发送应把胶囊并进正文最前并清空:%q %v", got, a.refs)
	}
	a.addRefs("@x.go")
	if got := a.foldRefs("  "); got != "@x.go" {
		t.Fatalf("正文为空时只发引用:%q", got)
	}
	if got := a.foldRefs("原样"); got != "原样" {
		t.Fatalf("没有胶囊应原样返回:%q", got)
	}
}
