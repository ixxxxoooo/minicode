package app

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// 「关闭」(⌘W / 文件 → 关闭)看你在哪(设计稿「窗口形态 · 关闭 ⌘W」):
//   - 在工作区的编辑器标签组里(IDE 的编辑区、Agent 形态右侧的工作区)→ 只关当前标签;
//   - 在底部面板的终端里 → 关当前终端标签(只剩一个时收起面板,shell 继续跑);
//   - 拆分后空着的那一组 → 关掉这一组;
//   - 其它地方(导航栏 / 对话栏 / 设置与整页功能),或者所在的标签组已经空了 → 关闭窗口。
// 关闭窗口不是退出程序:主窗口只隐藏(进程、会话、后台任务、终端照常),从属窗口(⇧⌘N)才是真关闭。

// closeZone 是「你在哪」:最近一次交互落在的区域。
type closeZone uint8

const (
	zoneOther  closeZone = iota // 导航栏 / 对话栏 / 整页功能……:没有多标签
	zoneEditor                  // 工作区的编辑器标签组
	zonePanel                   // 底部面板(终端标签)
)

// zoneTrack 记录「你在哪」。三个来源,后发生的覆盖先发生的:
// ① 指针按下 / 按键落在哪个区域(ObserveInput,捕获阶段,不吃事件);
// ② 键盘焦点换了区域(比如命令面板把焦点交给输入框);
// ③ 你刚点开 / 切到一个标签(文件树单击、会话流「查看」)——限定在最近一次输入之后很短的时间里,
// 所以 Claude 自己开的标签(计划自动打开等)不会改变你的位置。
type zoneTrack struct {
	zone    closeZone
	inputAt time.Time // 最近一次用户输入

	edFocus, panelFocus bool      // 本帧:键盘焦点在工作区 / 底部面板里(构建时量,见 zoneMark*)
	focus               closeZone // 上一帧键盘焦点所在区域
	focusKnown          bool      // 上一帧窗口里有键盘焦点

	seen bool   // 已经看过一帧(第一帧只记录,不算「刚点开」)
	tab  string // 上一帧焦点组的当前标签
	proj string // 上一帧的项目(换项目带来的标签变化不算)
}

// zoneOpenWindow 是「刚点开的标签算你过去了」的时间窗:点击(按下)到标签出现之间通常只有一两帧。
const zoneOpenWindow = 800 * time.Millisecond

// zoneObserve 给某个区域的根元素用:输入落在它里面就把位置记成 z。
// 外层先于内层收到(捕获顺序),所以 窗口根 → 工作区 → 底部面板 依次覆盖,最后留下最里面的那个。
func (a *App) zoneObserve(z closeZone) func(ui.InputEvent) {
	return func(ev ui.InputEvent) {
		switch ev.Kind {
		case ui.InputPointerDown, ui.InputKeyDown, ui.InputText:
			a.ide.cz.zone = z
			a.ide.cz.inputAt = a.now()
		}
	}
}

// zoneMarkEditor / zoneMarkPanel 在构建工作区 / 底部面板时调用:接上输入观察,并记下本帧焦点是否在里面。
func (a *App) zoneMarkEditor(el ui.Element) {
	el.ObserveInput(a.zoneObserve(zoneEditor))
	if el.FocusWithin() {
		a.ide.cz.edFocus = true
	}
}

func (a *App) zoneMarkPanel(el ui.Element) {
	el.ObserveInput(a.zoneObserve(zonePanel))
	if el.FocusWithin() {
		a.ide.cz.panelFocus = true
	}
}

// zoneBegin 每帧构建前清掉本帧的焦点标记。
func (a *App) zoneBegin() {
	a.ide.cz.edFocus, a.ide.cz.panelFocus = false, false
}

// zoneSync 每帧构建完调用:焦点换了区域、或者刚点开一个标签时,让位置跟过去。
// rootFocus = 窗口内容里有键盘焦点(弹层 / 菜单 / 对话框不在其中,那时保持原位置)。
func (a *App) zoneSync(rootFocus bool) {
	z := &a.ide.cz
	fz, known := zoneOther, rootFocus
	switch {
	case z.panelFocus: // 底部面板在工作区里面,先判它
		fz, known = zonePanel, true
	case z.edFocus:
		fz, known = zoneEditor, true
	}
	if known && (!z.focusKnown || fz != z.focus) {
		z.zone = fz
	}
	z.focus, z.focusKnown = fz, known

	cur := ""
	if t := a.ide.cur(); t != nil && a.wsShown() {
		cur = t.id
	}
	if z.seen && cur != "" && cur != z.tab && a.proj == z.proj && !z.inputAt.IsZero() && a.now().Sub(z.inputAt) < zoneOpenWindow {
		z.zone = zoneEditor
	}
	z.tab, z.proj, z.seen = cur, a.proj, true
}

// wsShown 工作区此刻是否在屏幕上(整页功能 / 设置盖住时不算;含窄窗下的浮层)。
func (a *App) wsShown() bool {
	if a.screen != ScreenWork && a.screen != ScreenWelcome {
		return false
	}
	return a.wsActive(true) || a.wsActive(false) || (a.useWsOverlay() && a.rightOn && a.wsFloat)
}

// closeKind 是一次「关闭」落到的动作。
type closeKind string

const (
	closeTab    closeKind = "标签"
	closeGroup  closeKind = "编辑器组"
	closeTermT  closeKind = "终端标签"
	closePanel  closeKind = "收起底部面板"
	closeWindow closeKind = "窗口"
)

// closeTarget 按「你在哪」决定这次关闭关什么(纯判断,不改状态)。
func (a *App) closeTarget() closeKind {
	d := a.ide
	if !a.wsShown() || d.cz.zone == zoneOther {
		return closeWindow
	}
	if d.cz.zone == zonePanel && d.bottomOpen && d.bottomTab == "term" {
		if len(d.tps) >= 2 {
			return closeTermT
		}
		return closePanel // 至少保留一个终端(与终端菜单一致):只收起面板,不结束 shell
	}
	// 编辑器标签组;底部面板的「问题 / 输出」分段不是标签,也按这里处理
	if a.ly.panelMax && d.bottomOpen {
		return closePanel // 面板最大化时标签条让位:看不见的标签不关,先把面板收起来
	}
	if d.cur() != nil {
		return closeTab
	}
	if d.splitDir != 0 {
		return closeGroup
	}
	return closeWindow
}

// closeHere 是 ⌘W / 文件 → 关闭:看你在哪,关标签、关终端标签,或者关窗口。
func (a *App) closeHere() closeKind {
	d := a.ide
	k := a.closeTarget()
	switch k {
	case closeTab:
		tb := d.cur()
		log.Info("关闭", "对象", k, "tab", tb.id, "组", d.paneSel, "未保存", a.closeLoses(tb))
		if a.closeLoses(tb) { // ⌘W 比点 × 容易误按:会丢修改时先确认
			a.confirmDlg("不保存就关闭?", "「"+a.tabTitle(tb)+"」有未保存的修改,关闭后这些修改会丢失。", "不保存并关闭", true, func() {
				if d.cur() == tb {
					a.closeCurTab()
				}
			})
			break
		}
		a.closeCurTab()
	case closeGroup:
		log.Info("关闭", "对象", k, "组", d.paneSel)
		if d.paneSel == 0 { // 空的是主组:把次组的标签换过来再合并
			d.tabs, d.active = d.tabs2, d.active2
		}
		a.mergeSplit()
	case closeTermT:
		log.Info("关闭", "对象", k, "index", d.tpSel)
		a.askCloseTermTab(d.tpSel)
	case closePanel:
		log.Info("关闭", "对象", k)
		d.bottomOpen = false
		d.cz.zone = zoneEditor // 面板没了,人回到编辑器标签组
	case closeWindow:
		a.closeWindowOnly("⌘W")
	}
	a.redraw()
	return k
}

// closeLoses 关掉这个标签会不会丢未保存的修改(同一份缓冲区还开在另一组里就不会)。
func (a *App) closeLoses(tb *ideTab) bool {
	if tb == nil || !tb.dirty() {
		return false
	}
	d := a.ide
	for _, o := range append(append([]*ideTab{}, d.tabs...), d.tabs2...) {
		if o != tb && o.buf == tb.buf {
			return false
		}
	}
	return true
}

// closeCurTab 关掉焦点组的当前标签(只关这一组里的;同一文件在另一组的标签留着)。
func (a *App) closeCurTab() {
	d := a.ide
	tb := d.cur()
	if tb == nil {
		return
	}
	if d.splitDir == 0 {
		a.closeWsTab(tb.id) // 没拆分:与点标签上的 × 同一条路
		return
	}
	if d.paneSel == 1 {
		d.closeOnPane(tb.id, &d.tabs2, &d.active2)
	} else {
		d.closeOnPane(tb.id, &d.tabs, &d.active)
	}
}

// closeWindowOnly 关闭窗口但不退出程序:主窗口只隐藏(会话、后台任务、终端照常,Dock / 托盘可回来);
// 从属窗口(⇧⌘N 起的进程)走正常关闭——它只管自己这一个窗口,别的窗口不受影响。
func (a *App) closeWindowOnly(why string) {
	if a.win == nil { // 测试 / 截图:没有窗口
		return
	}
	if a.secondary {
		log.Info("关闭窗口", "action", "关闭从属窗口", "来源", why)
		a.win.Close()
		return
	}
	log.Info("关闭窗口", "action", "隐藏", "来源", why)
	a.win.Hide()
}
