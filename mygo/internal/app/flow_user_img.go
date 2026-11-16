package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// imgExist 缓存「图片文件是否存在」的判断(界面每帧都会重建会话流,
// 每条带图片的用户消息每帧都 os.Stat 一次会在界面线程上累计阻塞 IO,长会话里十几张图 × 每帧)。
// 结果缓存 imgExistTTL,过期后下一次访问再查;文件之后出现 / 消失最多延迟这么久才反映到界面上。
var imgExist = struct {
	sync.Mutex
	m map[string]imgExistEntry
}{m: map[string]imgExistEntry{}}

type imgExistEntry struct {
	ok bool
	at time.Time
}

const imgExistTTL = 3 * time.Second

func fileExistsCached(path string) bool {
	now := time.Now()
	imgExist.Lock()
	e, hit := imgExist.m[path]
	imgExist.Unlock()
	if hit && now.Sub(e.at) < imgExistTTL {
		return e.ok
	}
	_, err := os.Stat(path)
	e = imgExistEntry{ok: err == nil, at: now}
	imgExist.Lock()
	imgExist.m[path] = e
	imgExist.Unlock()
	return e.ok
}

func (a *App) resolveUserImage(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		if fileExistsCached(p) {
			return p
		}
		return ""
	}
	if strings.HasPrefix(p, "/demo/") {
		rel := strings.TrimPrefix(p, "/demo/")
		if a.ide != nil && a.ide.fs != nil {
			abs := a.ide.fs.Abs(rel)
			if fileExistsCached(abs) {
				return abs
			}
		}
	}
	if a.proj != "" {
		abs := filepath.Join(a.proj, p)
		if fileExistsCached(abs) {
			return abs
		}
	}
	return ""
}

// 缩略图与预览的尺寸(DIP)。
const (
	userThumbH    = 64  // 缩略图固定高度
	userThumbMinW = 64  // 缩略图最小宽(竖图 / 方图)
	userThumbMaxW = 112 // 缩略图最大宽(宽截图按比例裁到这里)
	userPreviewW  = 360 // 展开预览的最大宽
	userPreviewH  = 280 // 展开预览的最大高
)

// thumbSize 按原图比例给缩略图定宽:高固定,宽夹在 [minW, maxW],超出部分居中裁剪。
func thumbSize(src ui.ImageSource) (w, h float32) {
	iw, ih := ui.ImageSize(src)
	if iw <= 0 || ih <= 0 {
		return userThumbMinW, userThumbH
	}
	w = userThumbH * iw / ih
	if w < userThumbMinW {
		w = userThumbMinW
	}
	if w > userThumbMaxW {
		w = userThumbMaxW
	}
	return w, userThumbH
}

// previewSize 把原图等比缩到预览上限内(不放大)。
func previewSize(src ui.ImageSource) (w, h float32) {
	iw, ih := ui.ImageSize(src)
	if iw <= 0 || ih <= 0 {
		return userPreviewW, userPreviewH
	}
	k := float32(1)
	if iw > userPreviewW {
		k = userPreviewW / iw
	}
	if ih*k > userPreviewH {
		k = userPreviewH / ih
	}
	return iw * k, ih * k
}

// userMsgImages 用户消息里的图片:文字下方一排整齐的缩略图,点哪张就在下方展开哪张的预览
// (再点或「收起」关闭)。占位标记「[Image #N]」已在文字里去掉,图片只在这里出现。
func (a *App) userMsgImages(c *ui.Context, u *core.User) {
	if len(u.Images) == 0 {
		return
	}
	t := a.tk
	expKey := func(i int) string { return "userimg:" + u.ID + ":" + itoa(i) }
	top := float32(8)
	if userDisplayText(u.Text) == "" {
		top = 0 // 只有图片:不留文字与图片之间的空隙
	}
	ui.Row(c).Wrap().Gap(6).Margin(top, 0, 0, 0).AlignItems(ui.Center).Children(func() {
		for i, raw := range u.Images {
			i := i
			abs := a.resolveUserImage(raw)
			if abs == "" {
				// 文件找不到:一枚带文件名的小胶囊,不占缩略图的大格子。
				ui.Row(c).Height(28).Padding(0, 10).Radius(t.RSm()).Background(t.Fill1).AlignItems(ui.Center).Children(func() {
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
			open := a.foldExpanded(expKey(i))
			btn := ui.ButtonBase(c).Key("uimg-"+u.ID+"-"+itoa(i)).Size(w, h).Radius(t.RSm()).
				Cursor(ui.CursorPointer).Background(t.Fill1)
			if open {
				btn.Border(1.5, t.Ac)
			} else {
				btn.Border(1, t.Bd1)
			}
			btn.OnClick(func() { a.setFold(expKey(i), !a.foldExpanded(expKey(i))) })
			btn.Children(func() {
				if src != nil {
					ui.Image(c, src).Size(w, h).Radius(t.RSm()).Fit(ui.Cover)
				} else {
					kit.Ic(c, "image", kit.S12).TextColor(t.Tx4)
				}
			})
		}
	})
	for i, raw := range u.Images {
		if !a.foldExpanded(expKey(i)) {
			continue
		}
		abs := a.resolveUserImage(raw)
		if abs == "" {
			continue
		}
		i := i
		ui.Column(c).Key("uimg-exp-"+u.ID+"-"+itoa(i)).Margin(8, 0, 0, 0).Gap(6).AlignItems(ui.Start).Children(func() {
			if src := a.localImage(abs); src != nil {
				w, h := previewSize(src)
				ui.Image(c, src).Size(w, h).Fit(ui.Contain).Radius(t.RMd()).Border(1, t.Bd1)
			}
			ui.Row(c).Gap(6).Children(func() {
				kit.Btn(c, "收起", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() { a.setFold(expKey(i), false) })
				kit.Btn(c, "在预览中打开", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "ext"}).OnClick(func() {
					_ = exec.Command("open", abs).Start()
				})
			})
		})
	}
}
