package app

import (
	"path"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

var ideLog = diag.L("ide")

// ideTab 是编辑区的一个标签:真文件,或某个文件的左右对比(HEAD ↔ 工作区)。
type ideTab struct {
	id      string // 文件 = 相对路径;对比 = "diff:" + 相对路径
	path    string // 相对项目根的路径
	diff    bool
	label   string // 对比标签的说明,如「HEAD ↔ 工作区」「a3f9c21 的改动」
	preview bool   // 预览标签(斜体,下一次单击会被替换)
	buf     *editor.Buffer
	st      *editor.State
	dv      *editor.DiffState
	saved   string  // 最近一次落盘的内容(判断是否有未保存修改)
	err     string  // 读取失败 / 太大时的提示
	reveal  *[3]int // 待选中的范围:行、起止列(下一次渲染时应用)
}

func (t *ideTab) dirty() bool { return !t.diff && t.buf != nil && t.buf.String() != t.saved }

func (t *ideTab) diffLabel() string {
	if t.label != "" {
		return t.label
	}
	return "HEAD ↔ 工作区"
}

func (t *ideTab) title() string {
	if t.diff {
		return path.Base(t.path) + " (" + t.diffLabel() + ")"
	}
	return path.Base(t.path)
}

// ide 是 IDE 形态的状态。与 Agent 形态共用同一个 App、同一份会话状态;
// 这里只放「只有 IDE 才有」的东西:编辑器标签、文件树展开状态、底部面板。
type ide struct {
	fs     workspace.FS
	git    map[string]string // 相对路径 → M / A
	headOf func(string) (string, bool)

	tabs   []*ideTab
	active string

	open map[string]bool // 文件树里展开的目录
	sel  string          // 文件树里选中的路径

	bottomOpen bool
	bottomTab  string // term / problems / output
	termLines  []termLine
	termSt     ui.ScrollState

	sections map[string]bool // 资源管理器区块折叠(大纲 / 时间线)

	filesCache []string                     // allFiles 的缓存
	listCache  map[string][]workspace.Entry // 目录列表缓存
}

// termLine 是终端面板里的一行(带色段)。
type termLine struct {
	segs []termSeg
}
type termSeg struct {
	text string
	kind byte // 0 默认 / p 成功 / e 错误 / d 暗 / b 粗
}

// newIDE 建立 IDE 状态;demo 时用沙盒项目。
func newIDE() *ide {
	d := &ide{
		open:       map[string]bool{"src": true, "tests": true},
		bottomOpen: true,
		bottomTab:  "term",
		sections:   map[string]bool{},
	}
	d.useFS(demo.Sandbox(), demo.GitStatus(), demo.HeadOf)
	d.termLines = demoTerm()
	return d
}

func (d *ide) useFS(fs workspace.FS, git map[string]string, head func(string) (string, bool)) {
	d.fs, d.git, d.headOf = fs, git, head
	d.tabs, d.active, d.sel = nil, "", ""
	d.filesCache, d.listCache = nil, nil
}

// tab 返回 id 对应的标签(没有则 nil)。
func (d *ide) tab(id string) *ideTab {
	for _, t := range d.tabs {
		if t.id == id {
			return t
		}
	}
	return nil
}

func (d *ide) cur() *ideTab { return d.tab(d.active) }

// openFile 打开文件。pin=false 打开预览标签(替换已有的预览标签),pin=true 固定。
func (d *ide) openFile(p string, pin bool) {
	if t := d.tab(p); t != nil {
		if pin {
			t.preview = false
		}
		d.active, d.sel = p, p
		return
	}
	end := diag.SpanSlow("ide", "打开文件", 50*time.Millisecond, "path", p)
	defer end()
	t := &ideTab{id: p, path: p, preview: !pin}
	src, err := d.fs.Read(p)
	if err != nil {
		t.err = "无法打开:" + err.Error()
		ideLog.Warn("读取文件失败", "path", p, "err", err)
	} else if strings.IndexByte(src, 0) >= 0 {
		t.err = "二进制文件,不能用文本方式打开"
	} else {
		t.buf = editor.NewBuffer(src)
		t.buf.SetLanguage(editor.LanguageFor(path.Base(p)))
		t.saved = src
		t.st = &editor.State{Buffer: t.buf, WrapGuides: nil}
		if head, ok := d.headOf(p); ok {
			t.st.DiffBase = editor.NewBuffer(head) // 行号栏的改动色条
		}
	}
	d.addTab(t)
}

// reloadOpen 把已打开且没有未保存修改的文件标签读回磁盘内容(批量替换之后用)。
func (d *ide) reloadOpen() {
	for _, t := range d.tabs {
		if t.diff || t.buf == nil || t.dirty() {
			continue
		}
		if src, err := d.fs.Read(t.path); err == nil && src != t.saved {
			t.buf.Reset(src)
			t.saved = src
		}
	}
}

// openAt 打开文件并(在下一次渲染时)选中 line 行 [col,end) 列的范围。
func (d *ide) openAt(p string, line, col, end int, pin bool) {
	d.openFile(p, pin)
	if t := d.tab(p); t != nil && t.buf != nil {
		t.reveal = &[3]int{line, col, end}
	}
}

// openDiff 打开某个文件的对比标签(HEAD ↔ 工作区)。
func (d *ide) openDiff(p string, pin bool) {
	id := "diff:" + p
	if t := d.tab(id); t != nil {
		if pin {
			t.preview = false
		}
		d.active = id
		return
	}
	head, ok := d.headOf(p)
	if !ok {
		head = ""
	}
	cur, err := d.fs.Read(p)
	if err != nil {
		cur = ""
	}
	d.addDiffTab(id, p, "", head, cur, pin)
}

// openRevDiff 打开某次提交里某个文件的对比标签(父提交 ↔ 该提交)。
func (d *ide) openRevDiff(short, p, old, nw string, pin bool) {
	id := "diff:rev-" + short + ":" + p
	if t := d.tab(id); t != nil {
		if pin {
			t.preview = false
		}
		d.active = id
		return
	}
	d.addDiffTab(id, p, short+" 的改动", old, nw, pin)
}

func (d *ide) addDiffTab(id, p, label, oldSrc, newSrc string, pin bool) {
	lang := editor.LanguageFor(path.Base(p))
	old, nw := editor.NewBuffer(oldSrc), editor.NewBuffer(newSrc)
	old.SetLanguage(lang)
	nw.SetLanguage(lang)
	t := &ideTab{id: id, path: p, diff: true, label: label, preview: !pin,
		dv: &editor.DiffState{Old: old, New: nw, ContextLines: 4}}
	d.addTab(t)
}

func (d *ide) addTab(t *ideTab) {
	if t.preview {
		for i, o := range d.tabs { // 替换已有预览标签,原位
			if o.preview {
				d.tabs[i] = t
				d.active, d.sel = t.id, t.path
				return
			}
		}
	}
	// 就近插入:放在当前标签右侧
	at := len(d.tabs)
	for i, o := range d.tabs {
		if o.id == d.active {
			at = i + 1
		}
	}
	d.tabs = append(d.tabs, nil)
	copy(d.tabs[at+1:], d.tabs[at:])
	d.tabs[at] = t
	d.active, d.sel = t.id, t.path
}

// closeTab 关闭标签;关掉的是当前标签时回到右邻(没有就左邻)。
func (d *ide) closeTab(id string) {
	for i, t := range d.tabs {
		if t.id != id {
			continue
		}
		d.tabs = append(d.tabs[:i], d.tabs[i+1:]...)
		if d.active == id {
			switch {
			case len(d.tabs) == 0:
				d.active = ""
			case i < len(d.tabs):
				d.active = d.tabs[i].id
			default:
				d.active = d.tabs[len(d.tabs)-1].id
			}
		}
		return
	}
}

// save 保存当前文件。
func (d *ide) save() error {
	t := d.cur()
	if t == nil || t.diff || t.buf == nil {
		return nil
	}
	end := diag.SpanSlow("ide", "保存文件", 100*time.Millisecond, "path", t.path)
	defer end()
	s := t.buf.String()
	if err := d.fs.Write(t.path, s); err != nil {
		ideLog.Error("保存失败", "path", t.path, "err", err)
		return err
	}
	t.saved = s
	t.buf.MarkSaved()
	ideLog.Info("已保存", "path", t.path, "bytes", len(s))
	return nil
}

// fileGit 返回文件在工作区的 Git 状态("" = 未改动)。
func (d *ide) fileGit(p string) string { return d.git[p] }

// dirGit 目录下有改动时返回 "M"(树里目录名染色用)。
func (d *ide) dirGit(dir string) string {
	for p := range d.git {
		if strings.HasPrefix(p, dir+"/") {
			return "M"
		}
	}
	return ""
}

// demoTerm 是设计稿里终端面板的内容。
func demoTerm() []termLine {
	prompt := func(cmd string) termLine {
		return termLine{[]termSeg{{"➜ ", 'p'}, {"router-demo ", 'b'}, {"git:(", 'd'}, {"main", 'e'}, {") ", 'd'}, {cmd, 0}}}
	}
	return []termLine{
		prompt("cargo test router"),
		{[]termSeg{{"    Finished", 'd'}, {" test profile in 1.92s", 0}}},
		{[]termSeg{{"test matches_ignoring_case ... ", 0}, {"ok", 'p'}}},
		{[]termSeg{{"test matches_trailing_slash ... ", 0}, {"ok", 'p'}}},
		{[]termSeg{{"test result: ", 0}, {"ok", 'p'}, {". 12 passed; 0 failed", 0}}},
		prompt(""),
	}
}

// seedDemo 摆出设计稿里 IDE 的初始标签:router.rs(当前)、router_test.rs(有未保存修改)、
// Cargo.toml(预览)、router.rs 的对比标签。
func (d *ide) seedDemo() {
	d.openFile("src/router.rs", true)
	d.openFile("tests/router_test.rs", true)
	if t := d.tab("tests/router_test.rs"); t != nil {
		t.saved = "// 磁盘上的版本\n" // 与缓冲区不同 = 有未保存修改
	}
	d.openFile("Cargo.toml", false)
	d.openDiff("src/router.rs", true)
	d.active = "src/router.rs"
	d.sel = "src/router.rs"
	if t := d.tab("src/router.rs"); t != nil && t.st != nil {
		t.st.GoTo(editor.Pos{Line: 16, Col: 57})
	}
}
