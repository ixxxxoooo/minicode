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

// readImagePath 返回 Read 工具读取的图片的本机绝对路径;不是读图片、读失败、文件已不存在时返回空串。
func (a *App) readImagePath(tc *core.ToolCall) string {
	if tc == nil || tc.Kind != core.KindRead || tc.Status != core.ToolCompleted {
		return ""
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
