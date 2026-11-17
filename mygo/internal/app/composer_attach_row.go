package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 输入框上方的附件行:粘贴 / 添加 / 拖入的图片是缩略图卡(不显示文件名),文件与文件夹是同高的文件卡
// (图标 + 名称 + 所在目录),两种卡共用一套圆角 / 描边 / 移除按钮,正文里不再出现路径文字。
//
// @author ixxxxoooo

const attachCardH = 56 // 附件卡高度(缩略图是它的正方形)

// composerAttachRow 画输入框顶部的附件行;没有附件时不占位。
func (a *App) composerAttachRow(c *ui.Context) {
	if len(a.draftImgs) == 0 && len(a.refs) == 0 {
		return
	}
	ui.Row(c).Padding(10, 10, 0, 10).Gap(8).Wrap().Children(func() {
		for _, tk := range a.draftImgs {
			for _, at := range parseDraftAttaches(tk) {
				a.attachImageCard(c, at.Path)
			}
		}
		for _, r := range a.refs {
			a.attachFileCard(c, r)
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

// attachFileCard 文件 / 文件夹附件(「@路径」引用):图标 + 名称 + 所在目录 + 移除;点击打开。
func (a *App) attachFileCard(c *ui.Context, ref string) {
	t := a.tk
	p := strings.TrimPrefix(ref, "@")
	isDir := strings.HasSuffix(p, "/")
	clean := strings.TrimSuffix(p, "/")
	name := baseName(clean)
	sub := "文件夹"
	if !isDir {
		sub = "文件"
	}
	if d := filepath.Dir(clean); d != "." && d != "/" && d != "" {
		sub = d
	}
	card := ui.Row(c).Key("ref-"+p).Height(attachCardH).MinWidth(0).MaxWidth(220).Padding(0, 4, 0, 10).Gap(8).Radius(t.RMd()).
		Border(1, t.Bd2).Background(t.Fill1).AlignItems(ui.Center).Shrink(0).Cursor(ui.CursorPointer)
	if card.Hovered() {
		card.Background(t.Fill2)
	}
	card.Tooltip(p)
	card.OnClick(func() { a.openPathRef(p) })
	card.Children(func() {
		if isDir {
			kit.Ic(c, "folder", kit.S16).TextColor(t.Tx2).Shrink(0)
		} else {
			a.fileIc(c, name)
		}
		ui.Column(c).Gap(1).MinWidth(0).Shrink(1).Children(func() {
			ui.Text(c, name).FontWeight(500).SingleLine()
			ui.Text(c, sub).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
		})
		kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "移除"}).OnClick(func() { a.removeRef(ref) })
	})
}
