package app

import (
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// @author ixxxxoooo

// 读取图片的步骤:参照 BaoCode,行下面直接显示这张图(默认展开,可收起),点图片打开全屏灯箱。

const (
	readImgMaxW = 480 // 缩略预览的最大宽(DIP)
	readImgMaxH = 320 // 缩略预览的最大高(DIP)
)

// readImagePath 返回 Read 工具读取的图片的本机绝对路径:先看工具结果里带回来的图片,再看原文件;不是读图片、读失败、都不存在时返回空串。
func (a *App) readImagePath(tc *core.ToolCall) string {
	if tc == nil || tc.Kind != core.KindRead || tc.Status != core.ToolCompleted {
		return ""
	}
	// 优先用工具结果里带回来的图片(翻译器已把 base64 落盘到临时目录):原文件被清掉(如 /tmp 下的截图)也能看
	for _, ct := range tc.Content {
		if ib, ok := ct.(*core.ImageBlock); ok && ib.Path != "" {
			if abs := a.resolveUserImage(ib.Path); abs != "" {
				return abs
			}
		}
	}
	p := inputStr(tc, "file_path")
	if p == "" && len(tc.Locations) > 0 {
		p = tc.Locations[0]
	}
	if p == "" || !isImageExt(strings.ToLower(filepath.Ext(p))) {
		return ""
	}
	return a.resolveUserImage(p)
}

// readImageCard 在读图步骤的展开体里画这张图:按原比例缩进 480×320 的框里,圆角 + 细边,点击打开灯箱;
// 还在解码时画一块带图片图标的占位,解码失败时给一行文字。参数 abs 是图片绝对路径。
func (a *App) readImageCard(c *ui.Context, abs string) {
	t := a.tk
	src := a.localImage(abs)
	if src == nil {
		ui.Row(c).Height(theme.Px(96)).Width(theme.Px(160)).Radius(t.RMd()).Background(t.Fill1).Border(1, t.Bd1).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			kit.Ic(c, "image", kit.S16).TextColor(t.Tx4)
		})
		return
	}
	iw, ih := ui.ImageSize(src)
	if iw <= 0 || ih <= 0 {
		ui.Text(c, "无法显示这张图片:"+baseName(abs)).FontSize(theme.FsCap).TextColor(t.Tx3)
		return
	}
	k := min(float32(readImgMaxW)/iw, float32(readImgMaxH)/ih, 1)
	w, h := iw*k, ih*k
	btn := ui.ButtonBase(c).Key("rdimg-"+abs).Size(w, h).Radius(t.RMd()).Cursor(ui.CursorPointer).Border(1, t.Bd1).Clip().Tooltip("点击查看")
	if btn.Hovered() {
		btn.Border(1, t.Bd3)
	}
	btn.OnClick(func() { a.openImageLightboxPath(abs) })
	btn.Children(func() {
		ui.Image(c, src).Size(w, h).Fit(ui.Contain)
	})
}

// imagesOf 收集一组工具调用里读取到的图片(本机绝对路径,按出现顺序,去重),折叠的分组行右边用缩略图显示。
func (a *App) imagesOf(tcs []*core.ToolCall) []string {
	var out []string
	seen := map[string]bool{}
	for _, tc := range tcs {
		if p := a.readImagePath(tc); p != "" && !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

// imagesOfItems 同 imagesOf,输入是时间线条目(回合折叠行用;条目里可能是工具调用或待确认的权限请求)。
func (a *App) imagesOfItems(items []core.Item) []string {
	var tcs []*core.ToolCall
	for _, it := range items {
		if v := doneToolOf(it); v != nil {
			tcs = append(tcs, v)
		}
	}
	return a.imagesOf(tcs)
}

const (
	stepThumbH   = 20 // 折叠行里缩略图的高(DIP)
	stepThumbMax = 8  // 最多显示几张,其余写「+N」
)

// stepThumbs 在折叠的分组行 / 回合行里,给里面读取到的每张图画一枚小缩略图(点击开灯箱,可左右翻看全部)。
func (a *App) stepThumbs(c *ui.Context, sp step) {
	t := a.tk
	ui.Row(c).Gap(4).Shrink(0).AlignItems(ui.Center).Margin(0, 0, 0, 2).Children(func() {
		for i, abs := range sp.thumbs {
			if i >= stepThumbMax {
				ui.Text(c, "+"+itoa(len(sp.thumbs)-stepThumbMax)).FontSize(theme.FsCap).TextColor(t.Tx4)
				break
			}
			src := a.localImage(abs)
			h := theme.Px(stepThumbH)
			w := h * 1.4
			if src != nil {
				if iw, ih := ui.ImageSize(src); iw > 0 && ih > 0 {
					w = min(max(h*iw/ih, h*1.1), h*2.2)
				}
			}
			idx := i
			btn := ui.ButtonBase(c).Key("sthumb-"+sp.id+"-"+itoa(i)).Size(w, h).Radius(4).Cursor(ui.CursorPointer).Background(t.Fill1).Border(1, t.Bd2).Clip().Tooltip("点击查看")
			if btn.Hovered() {
				btn.Border(1, t.Bd3)
			}
			btn.OnClick(func() {
				names := make([]string, len(sp.thumbs))
				for k, p := range sp.thumbs {
					names[k] = baseName(p)
				}
				a.openLightbox(sp.thumbs, names, idx)
			})
			btn.Children(func() {
				if src != nil {
					ui.Image(c, src).Size(w, h).Fit(ui.Cover)
				} else {
					kit.Ic(c, "image", kit.S12).TextColor(t.Tx4)
				}
			})
		}
	})
}
