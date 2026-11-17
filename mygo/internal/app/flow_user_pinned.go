package app

import (
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
	a.userPinned = append(a.userPinned, pinnedUserEdit{id: u.ID, draft: userDisplayText(u.Text), focus: true, at: kit.Now()})
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
		a.renderPinnedUserThumbs(c, u)
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
		edKey := "upin-ed-" + id
		kit.VScroll(c).Key("upin-scroll-" + id).MaxHeight(maxBody).Children(func() {
			ed := ui.TextArea(c.Key(edKey), draft).Lines(3, maxLines).FontSize(fs).LineHeight(lh).
				Background(ui.Transparent).BorderWidth(0).Padding(0).FocusRing(false).TextColor(t.Tx1)
			if pe := a.pinnedEditByID(id); pe != nil && pe.focus {
				ed.AutoFocus()
				pe.focus = false
			}
			ed.OnShortcut(0, ui.KeyEscape, func() { a.closePinnedUserEdit(id) })
			ed.OnShortcut(0, ui.KeyEnter, func() { a.submitPinnedUserEdit(id) })
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

func (a *App) renderPinnedUserThumbs(c *ui.Context, u *core.User) {
	if len(u.Images) == 0 {
		return
	}
	t := a.tk
	ui.Row(c).Wrap().Gap(6).AlignItems(ui.Center).Children(func() {
		for i, raw := range u.Images {
			i := i
			abs := a.resolveUserImage(raw)
			if abs == "" {
				ui.Row(c).Height(theme.Px(28)).Padding(0, 10).Radius(t.RSm()).Background(t.Fill1).AlignItems(ui.Center).Children(func() {
					a.fileIc(c, baseName(raw))
					ui.Text(c, baseName(raw)).FontSize(theme.FsMicro).TextColor(t.Tx4).Margin(0, 0, 0, 6)
				})
				continue
			}
			src := a.localImage(abs)
			w, h := float32(userThumbMinW), float32(userThumbH)
			if src != nil {
				w, h = thumbSize(src)
			}
			btn := ui.ButtonBase(c).Key("upin-img-"+u.ID+"-"+itoa(i)).Size(w, h).Radius(t.RSm()).
				Cursor(ui.CursorPointer).Background(t.Fill1).Border(1, t.Bd1).Tooltip("点击预览")
			btn.OnClick(func() { a.openUserImageLightbox(u, i) })
			btn.Children(func() {
				if src != nil {
					ui.Image(c, src).Size(w, h).Radius(t.RSm()).Fit(ui.Cover)
				} else {
					kit.Ic(c, "image", kit.S12).TextColor(t.Tx4)
				}
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
	if len(u.Images) > 0 {
		thumbs = float32(userThumbH) + 6
	}
	return body + thumbs + 62
}
