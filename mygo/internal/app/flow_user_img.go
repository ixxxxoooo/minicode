package app

import (
	"os"
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

// 缩略图尺寸(DIP)。
const (
	userThumbH    = 64
	userThumbMinW = 64
	userThumbMaxW = 112
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

// userMsgImages 用户消息里的图片:文字下方一排缩略图,点击打开全屏灯箱。
func (a *App) userMsgImages(c *ui.Context, u *core.User) {
	if len(u.Images) == 0 {
		return
	}
	t := a.tk
	top := float32(8)
	if userDisplayText(u.Text) == "" {
		top = 0
	}
	ui.Row(c).Wrap().Gap(6).Margin(top, 0, 0, 0).AlignItems(ui.Center).Children(func() {
		for i, raw := range u.Images {
			i := i
			abs := a.resolveUserImage(raw)
			if abs == "" {
				chip := ui.Row(c).Height(theme.Px(28)).Padding(0, 10).Radius(t.RSm()).Background(t.Fill1).AlignItems(ui.Center).Children(func() {
					a.fileIc(c, baseName(raw))
					ui.Text(c, baseName(raw)).FontSize(theme.FsMicro).TextColor(t.Tx4).Margin(0, 0, 0, 6)
				})
				_ = chip
				continue
			}
			src := a.localImage(abs)
			w, h := float32(userThumbMinW), float32(userThumbH)
			if src != nil {
				w, h = thumbSize(src)
			}
			btn := ui.ButtonBase(c).Key("uimg-"+u.ID+"-"+itoa(i)).Size(w, h).Radius(t.RSm()).
				Cursor(ui.CursorPointer).Background(t.Fill1).Border(1, t.Bd1).Tooltip("点击预览")
			if btn.Hovered() {
				btn.Background(t.Fill2)
			}
			if btn.Pressed() {
				btn.Background(t.Fill3)
			}
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
