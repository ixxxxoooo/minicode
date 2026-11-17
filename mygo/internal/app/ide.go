package app

import (
	"errors"
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
	dvOld   string // 差异标签上次载入的旧侧 / 新侧(refreshDiffTabs 用来判断工作区一侧有没有被用户改过)
	dvNew   string
	label   string // 对比标签的说明,如「HEAD ↔ 工作区」「a3f9c21 的改动」
	preview bool   // 预览标签(斜体,下一次单击会被替换)
	pinned  bool   // 固定标签:缩成图标宽、排在最左、「关闭其他」不关
	buf     *editor.Buffer
	st      *editor.State
	dv      *editor.DiffState
	imgd    *imgDiffState // 图片 / 二进制文件的差异:不逐行对比,改成并排预览(ide_imgdiff.go)
	saved   string        // 最近一次落盘的内容(已把 \r\n 归一成 \n,与缓冲区同口径;判断是否有未保存修改)
	eol     string        // 保存时使用的行尾(空 = LF;状态栏可切换,ide_eol.go)
	eolDisk string        // 磁盘上现在的行尾
	err     string        // 读取失败 / 太大时的提示
	reveal  *[3]int       // 待选中的范围:行、起止列(下一次渲染时应用)
	headKey string        // 上次用于 DiffBase 的 HEAD 文本摘要(ide_gutter.go)
	virt    string        // 虚拟标签的种类(review / plan / sub / task / conflict),空 = 文件或对比
	ref     string        // 虚拟标签引用的对象 id(子 Agent / 任务 id、冲突文件路径)

	lay     int8      // 差异视图:0 跟随偏好 scm.diff,1 左右,2 统一(用户在标签上点过的选择,窄档只是临时强制统一、不改它)
	dk      string    // 差异来源:head / work / staged / timeline(空 = 其它提交差异)
	revHash string    // 时间线预览对比对应的提交(完整哈希)
	hooked  bool      // 已接上悬停 / 转到定义 / 补全
	link    *edLink   // ⌘ 悬停时指针下可跳转的单词(ide_link.go)
	diagSet bool      // 演示诊断已装
	diagGen int       // 已铺到编辑器的语言服务诊断代数(+1)
	fq      *string   // 编辑器查找条的查询词(迷你地图画命中刻度)
	fcs     *bool     // 查找是否区分大小写
	mm      *mmInfo   // 迷你地图叠加数据缓存
	mmY     float64   // 上次看到的滚动位置(变了 = 正在滚动)
	mmAt    time.Time // 最近一次滚动的时刻:其后 3 秒内迷你地图显示视口滑块与行范围
	mmInit  bool      // mmY 已初始化(打开标签时恢复的滚动位置不算「滚动」)

	asVer      uint64    // 自动保存:上次看到的缓冲区版本
	asAt       time.Time // 自动保存:版本变化的时间
	blameKey   string    // 行内追溯当前显示的内容(去重,避免每帧 SetInlays)
	blameShown bool
	inlayVer   uint64 // 内联提示已应用到哪一版缓冲区

	viewer string       // 非文本查看器的种类:image / table(空 = 文本标签),见 ide_viewer.go
	vw     *viewerState // 查看器状态(载入结果、缩放、当前工作表)

	md     *mdState // Markdown 所见即所得的状态(edit_md.go)
	mdMode int8     // Markdown 视图:0 跟随偏好,mdWysiwyg / mdSource = 在标签上手动选的

	extBanner  bool // 磁盘上文件已变且本地有未保存修改
	extKeep    bool // 用户选择保留编辑器版本
	extOrphan  bool // 文件在磁盘上已不存在
	extDeleted bool // 用户确认已删除
	diskRO     bool // 工作区外只读(语言服务器跳转开的标准库 / 依赖,不可保存)

	scopeHoverLn int    // 代码区悬停行(0 起),-1 = 无
	langPick     string // 手动语言(假文件名后缀,空 = 按路径)

	scopeDbgHover     bool   // -ed scope:hover 演示态
	scopeCacheVer     uint64 // 范围查询缓存(按缓冲区版本 + 光标行)
	scopeCacheLn      int
	scopeHoverLnCache int
	scopeCache        scopeSnap
}

func (t *ideTab) dirty() bool {
	return !t.diff && t.buf != nil && (t.buf.String() != t.saved || t.eolNow() != t.eolSaved())
}

// eolNow / eolSaved:当前选用的行尾 / 磁盘上的行尾(空 = LF)。改行尾本身也算未保存修改。
func (t *ideTab) eolNow() string {
	if t.eol == "" {
		return eolLF
	}
	return t.eol
}

func (t *ideTab) eolSaved() string {
	if t.eolDisk == "" {
		return eolLF
	}
	return t.eolDisk
}

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
	git    map[string]string    // 相对路径 → M / A / U / …
	dirGit map[string]dirGitAgg // 目录 → 后代改动聚合(ide_tree_dir_git.go)
	headOf func(string) (string, bool)

	tabs      []*ideTab
	linkMemo  linkMemo         // ⌘ 悬停可跳转的判定缓存(ide_link.go)
	linkProbe string           // 进行中的 LSP 跳转探测(路径:偏移)
	fileFolds map[string][]int // 文件折叠行号(关标签后再开恢复,ide_fold_logic.go)
	active    string
	tabs2     []*ideTab // 第二编辑器组(拆分)
	active2   string
	splitDir  int8      // 0 无 / 1 向右 / 2 向下(ide_split.go)
	splitFrac float32   // 主组占比 0.2–0.8,默认 0.5
	paneSel   int8      // 当前键盘焦点在哪个组
	cz        zoneTrack // 「你在哪」:⌘W 关标签还是关窗口(close_ctx.go)

	open map[string]bool // 文件树里展开的目录
	sel  string          // 文件树里选中的路径

	bottomOpen bool
	onSaved    func(path string) // 保存成功后(语言服务 didSave)
	beforeSave func(*ideTab)     // 保存前(设置:保存时格式化 / 去行尾空白)
	onDiskSnap func(*ideTab)     // 打开 / 保存后记录磁盘快照(外部修改检测)
	closed     []string          // 最近关闭的文件标签(⇧⌘T 重新打开,最多 20 个)
	panelTop   float32           // 底部面板上缘(上一帧)
	bottomTab  string            // term / problems / output
	probScope  string            // 问题面板分段:open / ws
	probErr    bool              // 问题面板:仅显示错误
	termLines  []termLine
	termSt     ui.ScrollState
	cmds       []cmdView    // 「在终端面板查看」打开的只读命令标签
	termSel    string       // 当前终端分段选中的标签(空 = shell)
	tps        []*termPanel // 真实终端标签(真实模式),每个一个独立的 shell
	tpSel      int          // 当前终端标签
	lastActive string       // 自动保存(失焦):上一帧的活动标签
	termTabSt  ui.ScrollState
	termTabIx  int             // 标签条已滚入可见的终端下标
	termTabW   map[int]float32 // 各终端标签上一帧宽度
	panelW     float32         // 底部面板内容区宽度(上一帧)

	sections map[string]bool // 资源管理器区块折叠(大纲 / 时间线)
	rootShut bool            // 「文件夹」区块(文件树本体)是否已折叠
	revealed string          // 上次自动在文件树里定位的标签路径(ide_reveal.go)
	revealTo string          // 待滚入可见的树行路径(自动定位后一次性消费)
	fltOpen  bool            // 文件树过滤条已展开(ide_tree_filter.go)
	flt      string          // 过滤关键字
	fltKey   string          // 过滤结果缓存的键(关键字 + 文件清单版本)
	fltVis   map[string]bool // 过滤后应显示的路径(命中文件 + 祖先目录)
	fltN     int             // 命中的文件数
	treeSt   ui.ScrollState  // 文件树的滚动状态
	treeIdx  int             // 本帧已画的树行数(算目标行的内容位置)
	treeHit  int             // 目标行的行序号(-1 = 本帧没画到)
	treeH    float32         // 文件树视口高度(上一帧估算)
	tlMore   map[string]int  // 时间线:按文件多加载的条数上限
	tlIx     int             // 时间线键盘焦点行(-1 = 无)

	tabSt          ui.ScrollState     // 标签条的横向滚动
	tabSeen        string             // 已滚动到可见的标签
	midDown        string             // 鼠标中键按下时所在的标签 id(抬起仍在同一标签才关闭)
	tabW           map[string]float32 // 各标签上一帧的宽度
	nav            []navPos           // 导航历史(位置栈)
	ni             int                // 历史里当前项的下标
	refs           *refsState         // 底部「引用」分段的内容
	ov             ideOv              // 代码区浮层
	gotoLn         gotoLineUI         // 转到行浮条(ide_goto.go)
	blameLine      int
	blameHoverAt   time.Time           // 悬停追溯文字的起始时刻(满 300ms 才出卡)
	edViewH        float32             // 上一帧编辑区的高度(浮层判断下方放不放得下)
	rvBarBelow     bool                // 本帧行内审查条贴在光标行下方(提交卡要让开)
	blameCardShow  bool                // 提交卡已显示
	blameCardPin   bool                // 调试 -ed blamecard 钉住提交卡(不随鼠标移开收起,便于截图)
	blameCardHover bool                // 鼠标在提交卡上
	blameLeaveAt   time.Time           // 鼠标离开追溯文字 / 提交卡的起始时刻(满 250ms 才收起)
	textCache      map[string]string   // 读盘缓存(引用 / 定义索引用)
	symCache       map[string]symEntry // 符号缓存

	filesCache []string                     // allFiles 的缓存
	listCache  map[string][]workspace.Entry // 目录列表缓存

	diskSnaps   map[string]diskSnap // 已打开文件的磁盘快照(外部修改检测)
	extmodLast  time.Time
	extmodFocus bool
	extmodAlert map[string]diskSnap // 已提示过的外部修改(同一快照不再弹)
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
	d.useFS(workspace.NewWithExternal(demo.Sandbox()), demo.GitStatus(), demo.HeadOf)
	d.termLines = demoTermLinesFrom(demo.TermDemoLines())
	return d
}

func (d *ide) useFS(fs workspace.FS, git map[string]string, head func(string) (string, bool)) {
	d.fs, d.git, d.headOf = fs, git, head
	d.tabs, d.active, d.sel = nil, "", ""
	d.filesCache, d.listCache = nil, nil
	d.nav, d.ni, d.refs, d.ov = nil, 0, nil, ideOv{}
	d.invalidate()
}

// tab 返回 id 对应的标签(没有则 nil);两组编辑器都会找。
func (d *ide) tab(id string) *ideTab {
	for _, t := range d.tabs {
		if t.id == id {
			return t
		}
	}
	for _, t := range d.tabs2 {
		if t.id == id {
			return t
		}
	}
	return nil
}

func (d *ide) cur() *ideTab { return d.curOn(d.paneSel) }

func (d *ide) curOn(pane int8) *ideTab {
	if pane == 1 && d.splitDir != 0 {
		return d.tabOn(d.active2, d.tabs2)
	}
	return d.tabOn(d.active, d.tabs)
}

func (d *ide) tabOn(id string, tabs []*ideTab) *ideTab {
	for _, t := range tabs {
		if t.id == id {
			return t
		}
	}
	return nil
}

// tabByPath 按路径找首个非虚拟 / 非 diff 标签(恢复编辑器焦点用)。
func (d *ide) tabByPath(path string) *ideTab {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	for _, t := range d.tabs {
		if t != nil && t.virt == "" && !t.diff && t.path == path {
			return t
		}
	}
	for _, t := range d.tabs2 {
		if t != nil && t.virt == "" && !t.diff && t.path == path {
			return t
		}
	}
	return nil
}

func (d *ide) paneTabs(pane int8) []*ideTab {
	if pane == 1 && d.splitDir != 0 {
		return d.tabs2
	}
	return d.tabs
}

func (d *ide) paneActive(pane int8) *string {
	if pane == 1 && d.splitDir != 0 {
		return &d.active2
	}
	return &d.active
}

// openFile 打开文件。pin=false 打开预览标签(替换已有的预览标签),pin=true 固定。
func (d *ide) openFile(p string, pin bool) {
	if t := d.tab(p); t != nil {
		if pin {
			t.preview = false
		}
		d.active, d.sel = p, p
		if d.onDiskSnap != nil && t.buf != nil {
			d.onDiskSnap(t)
		}
		return
	}
	end := diag.SpanSlow("ide", "打开文件", 50*time.Millisecond, "path", p)
	defer end()
	t := &ideTab{id: p, path: p, preview: !pin}
	if k := viewerKind(p); k != "" { // 图片 / 表格:不当文本读,交给查看器(ide_viewer.go)
		t.viewer, t.vw = k, &viewerState{}
	} else {
		d.loadText(t)
	}
	d.addTab(t)
	if d.onDiskSnap != nil && t.buf != nil {
		d.onDiskSnap(t)
	}
}

// loadText 把标签的内容当文本读进缓冲区;读不出 / 是二进制时写 t.err(由 errView 显示并给出口)。
func (d *ide) loadText(t *ideTab) {
	p := t.path
	src, err := d.fs.Read(p)
	switch {
	case err != nil:
		t.err = "无法打开:" + friendlyReadErr(err)
		ideLog.Warn("读取文件失败", "path", p, "err", err)
	case strings.IndexByte(src, 0) >= 0:
		t.err = binaryHint(p)
	default:
		t.err = ""
		t.buf = editor.NewBuffer(src)
		if t.langPick != "" {
			t.buf.SetLanguage(editor.LanguageFor(t.langPick))
		} else {
			t.buf.SetLanguage(editor.LanguageFor(path.Base(p)))
		}
		t.saved = t.buf.String() // 缓冲区已把 \r\n 归一;用原文比较会让 CRLF 文件一打开就显示未保存
		t.eol = detectEOL(src)
		t.eolDisk = t.eol
		ro := workspace.IsExternalPath(p) && t.diskRO
		t.st = &editor.State{Buffer: t.buf, ReadOnly: ro, WrapGuides: nil}
		if head, ok := d.headOf(p); ok && !ro {
			t.st.DiffBase = editor.NewBuffer(head) // 行号栏的改动色条
		}
		d.restoreFolds(t)
	}
}

// reloadOpen 把已打开且没有未保存修改的文件标签读回磁盘内容(批量替换之后用)。
func (d *ide) reloadOpen() {
	for _, t := range d.tabs {
		if t.diff || t.buf == nil || t.dirty() {
			continue
		}
		if src, err := d.fs.Read(t.path); err == nil && normEOL(src) != t.saved {
			t.buf.Reset(src)
			t.saved = t.buf.String()
			t.eol, t.eolDisk = detectEOL(src), detectEOL(src)
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

// openTimelineDiff 在预览对比标签里打开文件历史(单击替换、双击固定;同一文件复用 diff:tl: 标签)。
func (d *ide) openTimelineDiff(hash, short, p, old, nw, label string, pin bool) {
	id := timelineDiffID(p)
	if t := d.tab(id); t != nil {
		d.applyDiffSnapshot(t, old, nw, label, hash, pin)
		d.active, d.sel = id, p
		return
	}
	d.addDiffTab(id, p, label, old, nw, pin)
	if t := d.tab(id); t != nil {
		t.dk, t.revHash = dkTimeline, hash
		t.dv.Editable = false
	}
}

// applyDiffSnapshot 把已打开的差异标签换成新的两侧内容与说明。
func (d *ide) applyDiffSnapshot(t *ideTab, old, nw, label, hash string, pin bool) {
	if pin {
		t.preview = false
	}
	md := diffMediaState(t.path, old, nw)
	if md != nil && !md.textOK {
		t.imgd = md
		old, nw = "", ""
	} else {
		t.imgd = nil
	}
	if t.dv == nil {
		return
	}
	t.dv.Old.Reset(old)
	t.dv.New.Reset(nw)
	t.dvOld, t.dvNew, t.label, t.revHash = old, nw, label, hash
	t.dk = dkTimeline
}

func (d *ide) addDiffTab(id, p, label, oldSrc, newSrc string, pin bool) {
	lang := editor.LanguageFor(path.Base(p))
	md := diffMediaState(p, oldSrc, newSrc)
	if md != nil && !md.textOK { // 图片 / 二进制:缓冲区留空,免得把字节当文本逐行算差异
		oldSrc, newSrc = "", ""
	}
	old, nw := editor.NewBuffer(oldSrc), editor.NewBuffer(newSrc)
	old.SetLanguage(lang)
	nw.SetLanguage(lang)
	t := &ideTab{id: id, path: p, diff: true, label: label, preview: !pin, imgd: md,
		dv: &editor.DiffState{Old: old, New: nw, ContextLines: 4}}
	d.addTab(t)
}

func (d *ide) addTab(t *ideTab) {
	d.addTabTo(d.paneSel, t)
}

func (d *ide) addTabTo(pane int8, t *ideTab) {
	tabs := &d.tabs
	active := &d.active
	if pane == 1 && d.splitDir != 0 {
		tabs, active = &d.tabs2, &d.active2
	}
	if t.preview {
		for i, o := range *tabs { // 替换已有预览标签,原位
			if o.preview {
				d.saveFolds(o)
				(*tabs)[i] = t
				*active, d.sel = t.id, t.path
				return
			}
		}
	}
	at := len(*tabs)
	for i, o := range *tabs {
		if o.id == *active {
			at = i + 1
		}
	}
	*tabs = append(*tabs, nil)
	copy((*tabs)[at+1:], (*tabs)[at:])
	(*tabs)[at] = t
	*tabs = reorderPinnedTabs(*tabs)
	*active, d.sel = t.id, t.path
}

// closeTab 关闭标签;关掉的是当前标签时回到右邻(没有就左邻)。
func (d *ide) closeTab(id string) {
	d.closeOnPane(id, &d.tabs, &d.active)
	d.closeOnPane(id, &d.tabs2, &d.active2)
}

func (d *ide) closeOnPane(id string, tabs *[]*ideTab, active *string) {
	for i, t := range *tabs {
		if t.id != id {
			continue
		}
		d.saveFolds(t)
		*tabs = append((*tabs)[:i], (*tabs)[i+1:]...)
		if isFileTab(t) && !t.diff && !(workspace.IsExternalPath(t.path) && t.diskRO) {
			d.closed = append(d.closed, t.path)
			if len(d.closed) > 20 {
				d.closed = d.closed[len(d.closed)-20:]
			}
		}
		if *active == id {
			switch {
			case len(*tabs) == 0:
				*active = ""
			case i < len(*tabs):
				*active = (*tabs)[i].id
			default:
				*active = (*tabs)[len(*tabs)-1].id
			}
		}
		return
	}
}

// save 保存当前文件。
func (d *ide) save() error {
	t := d.cur()
	if t != nil && t.diff && t.dv != nil && t.dv.Editable && t.dk != dkStaged { // 差异的工作区一侧:写回文件
		src := t.dv.New.String()
		if err := d.fs.Write(t.path, src); err != nil {
			return err
		}
		d.invalidate()
		d.reloadOpen()
		return nil
	}
	if t == nil || t.diff || t.buf == nil {
		return nil
	}
	if workspace.IsExternalPath(t.path) && t.diskRO {
		return errors.New("只读文件不能保存")
	}
	end := diag.SpanSlow("ide", "保存文件", 100*time.Millisecond, "path", t.path)
	defer end()
	if d.beforeSave != nil {
		d.beforeSave(t)
	}
	s := t.buf.String()
	if err := d.fs.Write(t.path, withEOL(s, t.eolNow())); err != nil {
		ideLog.Error("保存失败", "path", t.path, "err", err)
		return err
	}
	t.saved = s
	t.eolDisk = t.eolNow()
	t.buf.MarkSaved()
	if d.onSaved != nil {
		d.onSaved(t.path)
	}
	d.invalidate()
	if d.onDiskSnap != nil {
		d.onDiskSnap(t)
	}
	t.extBanner = false
	t.extKeep = false
	if !t.extOrphan {
		t.extDeleted = false
	}
	ideLog.Info("已保存", "path", t.path, "bytes", len(s))
	return nil
}

// fileGit 返回文件在工作区的 Git 状态("" = 未改动)。
func (d *ide) fileGit(p string) string { return d.git[p] }

func demoTermLinesFrom(in []demo.TermLine) []termLine {
	out := make([]termLine, len(in))
	for i, ln := range in {
		segs := make([]termSeg, len(ln.Segs))
		for j, s := range ln.Segs {
			segs[j] = termSeg{text: s.Text, kind: s.Kind}
		}
		out[i] = termLine{segs: segs}
	}
	return out
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
	d.active = "src/router.rs"
	d.sel = "src/router.rs"
	if t := d.tab("src/router.rs"); t != nil && t.st != nil {
		t.st.GoTo(editor.Pos{Line: 16, Col: 57})
	}
}

// cmdView 是终端面板里的一个只读命令标签:查看 Claude 跑过的某条命令及其输出。
type cmdView struct {
	id, title, output string
	exit              *int
}

// tpc 返回当前终端标签(没有就建一个,真正的 shell 要等面板量出大小才启动)。
func (d *ide) tpc() *termPanel {
	if len(d.tps) == 0 {
		d.tps = []*termPanel{{}}
		d.tpSel = 0
	}
	if d.tpSel < 0 || d.tpSel >= len(d.tps) {
		d.tpSel = len(d.tps) - 1
	}
	return d.tps[d.tpSel]
}

// fileTextBG 供后台线程读文件内容(只读磁盘,不碰缓存与缓冲区)。
func (d *ide) fileTextBG(p string) (string, bool) {
	s, err := d.fs.Read(p)
	if err != nil || len(s) > 2<<20 {
		return "", false
	}
	return s, true
}
