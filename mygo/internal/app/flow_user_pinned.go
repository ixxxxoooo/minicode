package app

import (
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui/richtext"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// pinnedUserEdit 是一条正在悬浮编辑的用户消息(可并存多条,后打开的盖在上面)。
type pinnedUserEdit struct {
	id    string
	draft string
	focus bool      // 刚打开:本帧自动聚焦
	at    time.Time // 打开的时刻:卡片从原来那张的大小「浮起并展开」的动画起点
	// doc 是编辑卡的富文本文档:正文里的 @文件 引用与输入框一样是行内胶囊;draft 始终是它展开后的发送文本。
	doc *richtext.Document
	ver uint64 // doc 上次同步回 draft 时的版本
	// 图片:与输入框一致,正文里是「图片 N」位置胶囊,缩略图在上方可移除、可粘贴新的(绝对路径,按序)。
	imgs       []string
	pendingImg []string        // 刚粘贴、等下一帧在光标处插胶囊的图片
	pilled     map[string]bool // 文档里现有的图片位置胶囊(路径集合):胶囊被删就移除对应图片
	rebuild    *string         // 移除缩略图后要按这段正文重建文档(建胶囊要 *ui.Context,点击回调里没有,留到下一帧画的时候做)
}

// userMsgAnchor 记录用户消息卡上一帧的位置(供悬浮编辑卡定位)。
type userMsgAnchor struct {
	rect ui.Rect
	row  int
}

// userCardClick 点击用户消息卡:已开则收起,否则打开悬浮编辑卡(吸顶与未吸顶一致)。
func (a *App) userCardClick(u *core.User, stuck bool) {
	_ = stuck
	if a.pinnedUserOpen(u.ID) {
		a.closePinnedUserEdit(u.ID)
	} else {
		a.openPinnedUserEdit(u)
	}
	a.frameSoon()
	log.Info("用户消息悬浮编辑", "user", u.ID, "open", a.pinnedUserOpen(u.ID))
}

func (a *App) pinnedUserOpen(id string) bool {
	for _, e := range a.userPinned {
		if e.id == id {
			return true
		}
	}
	return false
}

func (a *App) openPinnedUserEdit(u *core.User) {
	for i := range a.userPinned {
		if a.userPinned[i].id == u.ID {
			return
		}
	}
	var imgs []string
	for _, raw := range u.Images {
		if abs := a.resolveUserImage(raw); abs != "" {
			imgs = append(imgs, abs)
		}
	}
	a.userPinned = append(a.userPinned, pinnedUserEdit{id: u.ID, draft: pinnedDraftText(u), focus: true, at: kit.Now(), imgs: imgs})
}

func (a *App) closePinnedUserEdit(id string) {
	out := a.userPinned[:0]
	for _, e := range a.userPinned {
		if e.id != id {
			out = append(out, e)
		}
	}
	a.userPinned = out
}

func (a *App) cancelAllUserEdits() {
	a.userPinned = nil
}

func (a *App) cancelTopUserEdit() {
	if len(a.userPinned) == 0 {
		return
	}
	a.userPinned = a.userPinned[:len(a.userPinned)-1]
}

func (a *App) pinnedEditDraft(id string) *string {
	for i := range a.userPinned {
		if a.userPinned[i].id == id {
			return &a.userPinned[i].draft
		}
	}
	return nil
}

func (a *App) userByID(id string) *core.User {
	for _, it := range a.st.Items {
		if u, ok := it.(*core.User); ok && u.ID == id {
			return u
		}
	}
	return nil
}

// userPinnedOverlay 在会话流上方绘制悬浮编辑卡(不占位、不推挤列表)。
func (a *App) userPinnedOverlay(c *ui.Context) {
	if len(a.userPinned) == 0 {
		return
	}
	box := a.flowBox
	if box.W <= 0 || box.H <= 0 {
		return
	}
	// 绝对定位相对父容器(morphWrap 列,与列表同原点),不能用窗口坐标的 box.X / box.Y,
	// 否则对话栏不在窗口左上角时(IDE 右栏 / Agent 有导航栏)整层会被推出可视区
	layer := ui.Box(c).Absolute().Left(0).Top(0).Width(box.W).Height(box.H).ZIndex(8)
	layer.Children(func() {
		ui.ButtonBase(c).Absolute().Left(0).Right(0).Top(0).Bottom(0).ZIndex(0).
			OnClick(a.cancelAllUserEdits).Cursor(ui.CursorDefault)
		for i, pe := range a.userPinned {
			u := a.userByID(pe.id)
			if u == nil {
				continue
			}
			i := i
			anch, hasAnch := a.userAnchor[pe.id]
			top, left, w := a.pinnedEditPlacement(anch, hasAnch, box)
			ui.Box(c).Absolute().Left(left).Top(top).Width(w).ZIndex(10 + i).Children(func() {
				a.renderPinnedUserEdit(c, u, pe.id, w, box.H-top)
			})
		}
	})
}

func (a *App) pinnedEditPlacement(anch userMsgAnchor, hasAnch bool, box ui.Rect) (top, left, w float32) {
	pad := a.flowPad()
	colW := a.flowMaxWidth()
	w = colW - pad*2
	left = (box.W-colW)/2 + pad
	top = pinnedEditorInset
	if hasAnch {
		stuck := a.prefs.Bool(pStickyUser, true) && a.userStuck(anch.rect, anch.row)
		if stuck {
			// 吸顶的原卡贴在列表顶部的实际位置就是它的矩形(上一帧);用它,编辑卡才与原卡完全重合,
			// 不会比原卡低几像素、露出底下那一张的边
			top = pinnedEditorInset
			if y := anch.rect.Y - box.Y; y >= 0 && y < 40 {
				top = y
			}
		} else {
			top = anch.rect.Y - box.Y
			if top < pinnedEditorInset {
				top = pinnedEditorInset
			}
		}
		if aw := anch.rect.W; aw > 80 {
			w = min(aw, colW-pad*2)
			left = anch.rect.X - box.X
			if left < pad {
				left = pad
			}
			if left+w > box.W-pad {
				left = box.W - pad - w
			}
		}
	}
	return top, left, w
}

func (a *App) renderPinnedUserEdit(c *ui.Context, u *core.User, id string, cardW, below float32) {
	t := a.tk
	draft := a.pinnedEditDraft(id)
	if draft == nil {
		return
	}
	topY := float32(0)
	if anch, ok := a.userAnchor[id]; ok {
		_, topY, _ = a.pinnedEditPlacement(anch, ok, a.flowBox)
	}
	maxBody := pinnedEditorBodyMax(below+topY, topY)
	card := ui.Column(c).Key("upin-"+id).Background(t.Raised.Over(t.BG)).Radius(t.RMd()).
		Padding(9, 11, 7, 11).Gap(6).AlignItems(ui.Stretch).Border(1, t.AcLine).Width(cardW)
	for _, s := range t.Shadow2() {
		card.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	// 原地浮起并展开:这张卡就是原来那张(原卡此刻已隐藏,见 userMsg),起始高度 = 原卡高度,
	// 在 pinnedExpandDur 内长到编辑器的高度,而不是原卡还在、上面又弹出一张新的
	if pe := a.pinnedEditByID(id); pe != nil && kit.MotionOn() {
		if p := float32(kit.Since(pe.at)) / float32(pinnedExpandDur); p < 1 {
			a.frameSoon()
			if anch, ok := a.userAnchor[id]; ok && anch.rect.H > 0 {
				e := kit.EaseOut(max(p, 0))
				fin := a.pinnedEditEstimateH(u, *draft, maxBody)
				if fin > anch.rect.H {
					card.MaxHeight(anch.rect.H + (fin-anch.rect.H)*e).Clip()
				}
			}
		}
	}
	card.OnClick(func() {}) // 拦住点穿到背板
	card.Children(func() {
		a.renderPinnedUserThumbs(c, u, a.pinnedEditByID(id))
		fs := a.FlowFontSize()
		lh := a.FlowReadLH()
		lineH := fs * lh
		maxLines := int(maxBody / lineH)
		if maxLines < 3 {
			maxLines = 3
		}
		if maxLines > 32 {
			maxLines = 32
		}
		svc := c.Services() // 按键回调里没有 *ui.Context,读剪贴板走 Services
		pe := a.pinnedEditByID(id)
		a.pillImgPaths = pe.imgs // 建「图片 N」胶囊的序号按编辑卡自己的图片列表
		defer func() { a.pillImgPaths = nil }()
		if pe.doc == nil { // 第一次画:按草稿建文档,@路径 / [Image #N] 变成胶囊(与输入框同款)
			pe.doc = richtext.NewDocument(draftToText(userDraftBody(*draft, pe.imgs), func(kind, path string) *richtext.Attachment {
				return a.pillAttachment(c, kind, path)
			}))
			pe.ver = pe.doc.Version()
			pe.pilled = imgAtomPaths(textToDraft(pe.doc.Text()))
		}
		if pe.rebuild != nil { // 移除了某张图:按去掉它的位置胶囊后的正文重建(其余胶囊序号随之重排)
			pe.doc.SetText(draftToText(*pe.rebuild, func(kind, path string) *richtext.Attachment {
				return a.pillAttachment(c, kind, path)
			}))
			pe.pilled, pe.rebuild = imgAtomPaths(*pe.rebuild), nil
			a.syncPinnedDoc(pe)
		}
		doc := pe.doc
		for _, ip := range pe.pendingImg { // 刚粘贴的图:在光标处插位置胶囊
			doc.InsertAttachment(a.pillAttachment(c, atomImg, ip))
		}
		pe.pendingImg = nil
		if v := doc.Version(); v != pe.ver { // 用户改了:展开成发送文本,提交 / 复制 / 估高都用 *draft
			a.syncPinnedDoc(pe)
		}
		h := min(lineH*3, lineH*float32(maxLines))
		if boxes := doc.VisibleParagraphs(); len(boxes) > 0 {
			first, last := boxes[0], boxes[len(boxes)-1]
			h = min(max(h, last.Top+last.Height-first.Top), lineH*float32(maxLines))
		}
		th := &richtext.Theme{Text: t.Tx1, Muted: t.Tx3, Link: t.AcTx, Caret: t.Ac, Selection: t.Ac.Alpha(.22)}
		wrap := ui.Column(c.Key("upin-wrap-" + id)).Cursor(ui.CursorText)
		wrap.OnClick(func() { a.pillClickOpen(wrap, doc) }) // 点胶囊 = 打开它
		kit.VScroll(c).Key("upin-scroll-" + id).MaxHeight(maxBody).Children(func() {
			wrap.Children(func() {
				ed := richtext.EditorBase(c.Key("upin-ed-"+id), doc, richtext.EditorOptions{
					Label:      "编辑消息",
					Font:       ui.Font{Size: fs, LineHeight: lh},
					NoMarkdown: true,
					Theme:      th,
					Menu:       func(m *ui.Menu) {},
					OnKey: func(key ui.Key, mods ui.Modifiers) bool {
						switch {
						case key == ui.KeyV && mods == ui.Cmd: // ⌘V:图片 / 图片文件进这张卡(正文里插位置胶囊),文字走编辑器自己的粘贴
							return a.pinnedPaste(svc, id)
						case key == ui.KeyEscape && mods == 0:
							a.closePinnedUserEdit(id)
							return true
						case key == ui.KeyEnter && mods == 0:
							a.syncPinnedDoc(pe)
							a.submitPinnedUserEdit(id)
							return true
						}
						return false
					},
				}).Height(h)
				if pe.focus {
					ed.AutoFocus()
					pe.focus = false
				}
			})
		})
		ui.Box(c).Height(1).Background(t.Bd1)
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			hint := "Enter 发送 · Shift+Enter 换行 · Esc 关闭"
			if a.st.Phase == core.PhaseRunning {
				hint = "回合进行中:先结束或中断后再编辑发送"
			}
			ui.Text(c, hint).FontSize(theme.FsMicro).TextColor(t.Tx3).Grow(1).SingleLine()
			kit.Btn(c, "复制", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "copy"}).OnClick(func() {
				a.copyOut(strings.TrimSpace(*draft), "消息")
			})
			kit.Btn(c, "取消", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() { a.closePinnedUserEdit(id) })
			canReal := a.be == nil || a.be.Name() != "claude" || a.st.UserUUIDs[u.ID] != ""
			sub := kit.Btn(c, "发送", kit.BtnOpt{Small: true, Variant: kit.BtnPri, Icon: "send"})
			if canReal && a.st.Phase != core.PhaseRunning {
				sub.OnClick(func() { a.submitPinnedUserEdit(id) })
			} else {
				sub.Opacity(0.45).Cursor(ui.CursorDefault)
			}
		})
		if a.be != nil && a.be.Name() == "claude" && a.st.UserUUIDs[u.ID] == "" {
			ui.Text(c, "无检查点:无法 resume_session_at;请用「分叉会话」从该回合重试").FontSize(theme.FsMicro).TextColor(t.Tx4)
		}
	})
}

func (a *App) pinnedEditByID(id string) *pinnedUserEdit {
	for i := range a.userPinned {
		if a.userPinned[i].id == id {
			return &a.userPinned[i]
		}
	}
	return nil
}

func (a *App) renderPinnedUserThumbs(c *ui.Context, u *core.User, pe *pinnedUserEdit) {
	if pe == nil || len(pe.imgs) == 0 {
		return
	}
	t := a.tk
	ui.Row(c).Wrap().Gap(6).AlignItems(ui.Center).Children(func() {
		for i, abs := range pe.imgs {
			i, abs := i, abs
			src := a.localImage(abs)
			w, h := float32(userThumbMinW), float32(userThumbH)
			if src != nil {
				w, h = thumbSize(src)
			}
			btn := ui.ButtonBase(c).Key("upin-img-"+u.ID+"-"+itoa(i)).Size(w, h).Radius(t.RSm()).
				Cursor(ui.CursorPointer).Background(t.Fill1).Border(1, t.Bd1).Tooltip("点击预览")
			btn.OnClick(func() { a.openImageLightboxPath(abs) })
			btn.Children(func() {
				if src != nil {
					ui.Image(c, src).Size(w, h).Radius(t.RSm()).Fit(ui.Cover)
				} else {
					kit.Ic(c, "image", kit.S12).TextColor(t.Tx4)
				}
				a.attachRemoveBtn(c, "upin-img-x-"+u.ID+"-"+itoa(i), "移除", func() { a.removePinnedImage(pe, i) })
			})
		}
	})
}

// pinnedExpandDur 是悬浮编辑卡从原卡大小展开到编辑器大小的时长。
const pinnedExpandDur = 200 * time.Millisecond

// pinnedEditEstimateH 估算编辑卡展开后的高度(文字行数 × 行高 + 缩略图 + 分隔线与按钮行 + 内边距),动画终点用。
func (a *App) pinnedEditEstimateH(u *core.User, draft string, maxBody float32) float32 {
	lineH := a.FlowFontSize() * a.FlowReadLH()
	lines := max(estimatedUserLines(draft, 44), 3)
	body := min(float32(lines)*lineH, maxBody)
	thumbs := float32(0)
	if pe := a.pinnedEditByID(u.ID); pe != nil && len(pe.imgs) > 0 {
		thumbs = float32(userThumbH) + 6
	}
	return body + thumbs + 62
}

// userDraftBody 把用户消息文字里的 @文件 引用换成输入框草稿里的胶囊原子标记(其余原样),供编辑卡建文档。
func userDraftBody(txt string, imgs []string) string {
	var b strings.Builder
	last := 0
	imgK := 0
	for _, m := range userTokMatches(txt) {
		var path string
		switch {
		case m[10] >= 0: // [Image #N] → 图片位置胶囊(序号固定,路径是消息里第 k 张图)
			imgK++
			if imgK-1 < len(imgs) {
				b.WriteString(txt[last:m[0]])
				b.WriteString(atomString(atomImg, imgs[imgK-1]))
				last = m[1]
			}
			continue
		case m[6] >= 0:
			path = txt[m[6]:m[7]]
		case m[8] >= 0:
			path = txt[m[8]:m[9]]
		default:
			continue
		}
		b.WriteString(txt[last:m[0]])
		b.WriteString(refAtom("@" + path))
		last = m[1]
	}
	b.WriteString(txt[last:])
	return b.String()
}

// pinnedDraftText 编辑卡的初始文字:有图片时保留 [Image #N] 占位(在编辑器里画成胶囊),否则按显示文字。
func pinnedDraftText(u *core.User) string {
	if len(u.Images) > 0 {
		return strings.TrimSpace(u.Text)
	}
	return userDisplayText(u.Text)
}

// syncPinnedDoc 把编辑卡文档的当前内容展开成发送文本写回 draft;并处理「用户删了某张图的位置胶囊 → 移除这张图」。
func (a *App) syncPinnedDoc(pe *pinnedUserEdit) {
	if pe == nil || pe.doc == nil {
		return
	}
	body := textToDraft(pe.doc.Text())
	now := imgAtomPaths(body)
	var gone []string
	for p := range pe.pilled {
		if !now[p] {
			gone = append(gone, p)
		}
	}
	pe.pilled = now
	if len(gone) > 0 {
		var keep []string
		for _, ip := range pe.imgs {
			drop := false
			for _, g := range gone {
				drop = drop || ip == g
			}
			if !drop {
				keep = append(keep, ip)
			}
		}
		pe.imgs = keep
	}
	pe.draft = atomsToSendImgs(body, pe.imgs)
	pe.ver = pe.doc.Version()
}

// addPinnedImage 往编辑卡加一张图(粘贴 / 拖入):进缩略图行,并排队在光标处插位置胶囊;同一张不重复。
func (a *App) addPinnedImage(id, path string) bool {
	pe := a.pinnedEditByID(id)
	if pe == nil || path == "" {
		return false
	}
	for _, ip := range pe.imgs {
		if ip == path {
			return false
		}
	}
	pe.imgs = append(pe.imgs, path)
	pe.pendingImg = append(pe.pendingImg, path)
	return true
}

// pinnedPaste 编辑卡里 ⌘V:剪贴板有图片 / 图片文件就加进这张卡并返回 true;否则返回 false 交给编辑器粘贴文字。
func (a *App) pinnedPaste(svc ui.Services, id string) bool {
	a.pasteTo = id
	defer func() { a.pasteTo = "" }()
	if fs, err := svc.ReadClipboardData(transfer.FileList).Files(); err == nil && len(fs) > 0 {
		n := 0
		for _, f := range fs {
			switch strings.ToLower(filepath.Ext(f)) {
			case ".png", ".jpg", ".jpeg", ".gif", ".webp":
				if a.addDraftImage(f) {
					n++
				}
			}
		}
		return n > 0
	}
	return a.composerPasteImage(svc)
}

// removePinnedImage 移除编辑卡里第 i 张图;正文里它的位置胶囊在下一帧画的时候去掉(见 rebuild)。
func (a *App) removePinnedImage(pe *pinnedUserEdit, i int) {
	if pe == nil || i < 0 || i >= len(pe.imgs) {
		return
	}
	path := pe.imgs[i]
	pe.imgs = append(append([]string(nil), pe.imgs[:i]...), pe.imgs[i+1:]...)
	if pe.doc != nil {
		body := strings.ReplaceAll(textToDraft(pe.doc.Text()), atomString(atomImg, path), "")
		pe.rebuild = &body
	}
	a.frameSoon()
}
