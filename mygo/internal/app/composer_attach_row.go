package app

import (
	"os"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// 输入框上方的附件行:只有图片(粘贴 / 添加 / 拖入)是缩略图卡,不显示文件名,右上角移除钮,点开灯箱;
// 正文里不出现图片路径。文件 / 文件夹不在这里,它们是输入框正文里带样式的「@路径」(见 composer_refs.go)。
//
// @author ixxxxoooo

const attachCardH = 56 // 附件卡高度(缩略图是它的正方形)

// composerAttachRow 画输入框顶部的附件行;没有附件时不占位。
func (a *App) composerAttachRow(c *ui.Context) {
	if len(a.draftImgs) == 0 {
		return
	}
	ui.Row(c).Padding(10, 10, 0, 10).Gap(8).Wrap().Children(func() {
		for _, tk := range a.draftImgs {
			for _, at := range parseDraftAttaches(tk) {
				a.attachImageCard(c, at.Path)
			}
		}
	})
}

// attachRemoveBtn 缩略图右上角的小圆形移除钮。
func (a *App) attachRemoveBtn(c *ui.Context, key, tip string, fn func()) {
	t := a.tk
	b := ui.ButtonBase(c).Key(key).Absolute().Top(3).Right(3).Size(16, 16).Radius(8).Justify(ui.Center).AlignItems(ui.Center).
		Background(t.Tx1.Alpha(.7)).TextColor(t.BG).Cursor(ui.CursorPointer).Tooltip(tip)
	if b.Hovered() {
		b.Background(t.Tx1)
	}
	b.OnClick(fn)
	b.Children(func() { kit.Ic(c, "x", kit.S12) })
}

// attachImageCard 图片附件:正方形缩略图,点开灯箱预览;图读不出来时退回文件图标。
func (a *App) attachImageCard(c *ui.Context, rel string) {
	t := a.tk
	abs := a.absAttach(rel)
	card := ui.ButtonBase(c).Key("cmp-att-"+rel).Size(attachCardH, attachCardH).Radius(t.RMd()).Border(1, t.Bd2).Background(t.Fill1).
		Clip().Shrink(0).Justify(ui.Center).AlignItems(ui.Center).Cursor(ui.CursorPointer).Tooltip("点击预览")
	if abs != "" {
		card.OnClick(func() { a.openImageLightboxPath(abs) })
	}
	card.Children(func() {
		var src ui.ImageSource
		if abs != "" {
			if _, err := os.Stat(abs); err == nil {
				src = a.localImage(abs)
			}
		}
		if src != nil {
			ui.Image(c, src).Size(attachCardH, attachCardH).Fit(ui.Cover)
		} else {
			a.fileIc(c, baseName(rel))
		}
		a.attachRemoveBtn(c, "cmp-att-x-"+rel, "移除", func() { a.removeDraftAttachPath(rel) })
	})
}
