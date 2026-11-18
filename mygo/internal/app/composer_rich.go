package app

import (
	"strings"

	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
)

// 会话输入框的编辑器:富文本编辑器(richtext)承载正文,正文里的文件 / 文件夹 / 折叠文本是行内胶囊(原子)。
// 只当「纯文本 + 胶囊」用:关掉 Markdown 快捷键,粘贴一律自己接管成纯文本,所以不会混进粗体 / 标题。
//
// @author ixxxxoooo

// composerEditor 画输入框的编辑区,返回编辑器元素。
// 参数:svc 构建界面时取的 c.Services()(按键回调里读剪贴板用)。
func (a *App) composerEditor(c *ui.Context, svc ui.Services) ui.Element {
	t := a.tk
	doc := a.compDoc
	fs, lh := a.FlowFontSize(), a.FlowReadLH()
	line := fs * lh
	th := &richtext.Theme{
		Text: t.Tx1, Muted: t.Tx3, Link: t.AcTx,
		Caret: t.Ac, Selection: t.Ac.Alpha(.22),
	}
	// 编辑器高度:单行起步,内容多了按量到的末段底边长高,最多 8 行,再多就在里面滚
	// 不能直接用末段底边:编辑器太矮时会为露出光标把内容滚上去,底边就恒等于视口底、永远长不高。
	// 段落的 Top 是滚动后的坐标、Height 是整段的高,所以「首个可见段顶 → 末段底」的跨度与滚动无关;
	// 首个可见段不是第 0 段,说明上面还有内容被滚走了,直接给到最大高度。
	h, maxH := line, line*8
	if boxes := doc.VisibleParagraphs(); len(boxes) > 0 {
		first, last := boxes[0], boxes[len(boxes)-1]
		h = max(h, last.Top+last.Height-first.Top)
		if first.Index > 0 {
			h = maxH
		}
	}
	h = min(h, maxH)
	var ed ui.Element
	if a.ux.clipHave { // 上一次接管粘贴时临时清空了剪贴板,把原内容写回
		svc.WriteClipboardData(a.ux.clipRestore)
		a.ux.clipHave = false
	}
	// 编辑器自己不带内边距,套一层容器补上(上 9 下 3 左右 12,与原来的文本区一致),点内边距也能聚焦
	wrap := ui.Column(c.Key("draft-wrap")).Padding(9, 12, 3, 12).MinHeight(36).Cursor(ui.CursorText)
	// 真实应用里 ⌘V 多半不是按键事件,而是菜单的「粘贴」命令直接交给编辑器(OnKey 看不到),
	// 所以在外层先看一眼(捕获阶段、先于编辑器):剪贴板里有文件 / 图片 / 文本就自己处理
	wrap.ObserveInput(func(ev ui.InputEvent) {
		if ev.Kind == ui.InputCommand && (ev.Text == "paste" || ev.Text == "pasteAndMatchStyle") {
			a.composerPasteCommand(svc)
		}
	})
	wrap.Children(func() {
		// 编辑器是自绘的,滚动后会画出自己的矩形(盖住上面的缩略图行),所以套一层不带内边距的裁剪容器
		ui.Column(c.Key("draft-clip")).Clip().Children(func() {
			ed = richtext.EditorBase(c.Key("draft"), doc, richtext.EditorOptions{
				Label:       "输入消息",
				Placeholder: a.composerHint(),
				Font:        ui.Font{Size: fs, LineHeight: lh},
				NoMarkdown:  true,
				Theme:       th,
				Menu:        func(m *ui.Menu) {}, // 不给系统的富文本菜单:它的「粘贴」会绕过上面的纯文本 / 图片 / 文件处理
				OnKey: func(key ui.Key, mods ui.Modifiers) bool {
					return a.composerKey(svc, key, mods)
				},
			}).Height(h)
		})
	})
	if wrap.Clicked() {
		a.ux.focusDraft = true
	}
	wrap.OnClick(func() { a.pillClickOpen(wrap, doc) }) // 点胶囊 = 打开它
	a.flushPendingPills(c)
	return ed
}

// composerKey 输入框按键:⌘V 自己接管;斜杠补全优先;Enter / ⌘Enter 按偏好发送。返回 true = 已处理。
func (a *App) composerKey(svc ui.Services, key ui.Key, mods ui.Modifiers) bool {
	if mods == ui.Cmd && key == ui.KeyV {
		return a.composerPaste(svc)
	}
	if a.slashKey(key, mods) {
		return true
	}
	if key != ui.KeyEnter || (mods != 0 && mods != ui.Cmd) {
		return false
	}
	cmdSend := a.prefs.Int(pSend, 0) == 1 // 「⌘ Enter 发送」:Enter 只换行
	if cmdSend && mods == 0 {
		return false
	}
	// 回合进行中:发送键按「再发消息」偏好排队或引导;另一个修饰键(Enter 发送模式下的 ⌘Enter)取相反
	steer := a.prefs.Int(pBusy, 0) == 1
	if !cmdSend && mods == ui.Cmd {
		steer = !steer
	}
	txt := a.draftNow()
	a.submitKey(txt, steer)
	a.draft = "" // 发完 / 排队后清空输入(下一帧编辑器随之清空)
	return true
}

// composerPaste ⌘V:文件(访达复制)→ 胶囊 / 图片附件;剪贴板图片 → 图片附件;长文本 → 折叠胶囊;其余按纯文本插入。
// 返回 true = 已接管;false = 剪贴板里没有可用内容,交给编辑器。
func (a *App) composerPaste(svc ui.Services) bool {
	if fs, err := svc.ReadClipboardData(transfer.FileList).Files(); err == nil && len(fs) > 0 {
		a.handleDroppedImages(fs)
		return true
	}
	if a.composerPasteImage(svc) {
		return true
	}
	text := svc.ReadClipboard()
	if text == "" {
		return false
	}
	if a.HandleComposerPaste(text) {
		return true
	}
	a.compDoc.InsertText(text)
	return true
}

// composerPasteCommand 处理菜单「粘贴」命令:先于编辑器看到它,自己接管后把系统剪贴板临时清空,
// 编辑器随后的原生粘贴就什么也粘不进去(不会重复,也不会混进富文本格式),下一帧再把剪贴板原样写回。
func (a *App) composerPasteCommand(svc ui.Services) {
	orig := svc.ReadClipboardData(svc.ClipboardFormats()...).Snapshot()
	if !a.composerPaste(svc) {
		return
	}
	a.ux.clipRestore, a.ux.clipHave = orig, true
	svc.WriteClipboardData(transfer.New(transfer.NewItem(transfer.Bytes(transfer.Text, nil))))
	attachLog.Info("接管粘贴命令", "formats", len(svc.ClipboardFormats()))
}

// queuePill 排队一枚胶囊,下一帧在光标处插入(按键 / 拖放回调里没有 *ui.Context,量字要等构建期)。
// 返回:是否真的排上了(同一个文件 / 文件夹已在输入框里或已排队则不重复)。
func (a *App) queuePill(kind, path string) bool {
	d := pillData{Kind: kind, Path: path}
	if kind == atomFile || kind == atomDir || kind == atomImg {
		for _, q := range a.ux.pendingPills {
			if q == d {
				return false
			}
		}
		if at := atomString(kind, path); a.compDoc != nil && strings.Contains(textToDraft(a.compDoc.Text()), at) {
			return false
		}
	}
	a.ux.pendingPills = append(a.ux.pendingPills, d)
	a.ux.focusDraft = true
	return true
}

// flushPendingPills 把排队的胶囊插进光标处(每枚一步撤销)。
func (a *App) flushPendingPills(c *ui.Context) {
	for _, d := range a.ux.pendingPills {
		a.compDoc.InsertAttachment(a.pillAttachment(c, d.Kind, d.Path))
	}
	a.ux.pendingPills = nil
}
