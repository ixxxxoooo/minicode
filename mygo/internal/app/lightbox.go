package app

import (
	"math"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

var lbLog = diag.L("app.lightbox")

var lbInk = ui.Color{R: 255, G: 255, B: 255, A: 255}

// lbScrim 是灯箱的遮罩:近乎不透明的深色(约 93%),图片周围干净、不被背后的界面干扰。
// (MyGo 没有「背景模糊」滤镜,只能模糊元素自身,所以用更实的遮罩而不是半透明。)
var lbScrim = ui.Color{R: 8, G: 8, B: 10, A: 238}

// 灯箱出现 / 消失的动画时长:遮罩与内容一起淡入,图片从 94% 缓缓放大到位;关闭时反向、更快。
const (
	lbOpenDur  = 280 * time.Millisecond
	lbCloseDur = 160 * time.Millisecond
)

func lbInkA(a float32) ui.Color { return lbInk.Alpha(a) }

// lightboxState 当前打开的灯箱(nil 表示关闭)。
type lightboxState struct {
	paths      []string // 绝对路径
	names      []string // 显示名
	idx        int
	zoom       float32
	offX, offY float32
	natW, natH float32
	loaded     bool
	miss       bool // 当前图读不到
	drag       *lbDrag
	at         time.Time // 打开时刻(入场动画起点)
	closeAt    time.Time // 开始关闭的时刻(零值 = 没在关);关闭动画结束后才真正销毁
	aw, ah     float32   // 图片区域上一帧的宽高:滚轮缩放要把光标换算成相对区域中心的位置
}

type lbDrag struct {
	sx, sy, ox, oy float32
	moved          bool
}

func (a *App) openLightbox(paths []string, names []string, idx int) {
	paths = append([]string(nil), paths...)
	names = append([]string(nil), names...)
	if len(paths) == 0 {
		return
	}
	if idx < 0 || idx >= len(paths) {
		idx = 0
	}
	for len(names) < len(paths) { // 显示名少于图片数时补齐(否则按序号取名会越界)
		names = append(names, "")
	}
	for i := range names {
		if names[i] == "" {
			names[i] = baseName(paths[i])
		}
	}
	lb := &lightboxState{paths: paths, names: names, idx: idx, zoom: 1, at: kit.Now()}
	a.lb = lb
	a.lbRefreshNatural()
	a.frameSoon()
	lbLog.Info("打开灯箱", "path", paths[idx], "n", len(paths), "idx", idx)
}

// closeLightbox 关闭灯箱:有动效时先播 lbCloseDur 的淡出再销毁(重复调用不重置),没有动效立刻销毁。
func (a *App) closeLightbox() {
	if a.lb == nil {
		return
	}
	if !kit.MotionOn() {
		lbLog.Info("关闭灯箱")
		a.lb = nil
		return
	}
	if a.lb.closeAt.IsZero() {
		lbLog.Info("关闭灯箱")
		a.lb.closeAt = kit.Now()
		a.frameSoon()
	}
}

// lbProgress 返回入场 / 退场进度:p 是遮罩与内容的不透明度(0..1),s 是图片缩放系数。
// 入场 ease-out(先快后慢,落位柔和),退场更快;动效关闭时恒为 1。
func (lb *lightboxState) lbProgress() (p, s float32) {
	if !kit.MotionOn() {
		return 1, 1
	}
	if !lb.closeAt.IsZero() {
		q := float32(kit.Since(lb.closeAt)) / float32(lbCloseDur)
		q = min(max(q, 0), 1)
		return 1 - kit.EaseOut(q), 1 - .03*kit.EaseOut(q)
	}
	e := kit.EaseOut(min(max(float32(kit.Since(lb.at))/float32(lbOpenDur), 0), 1))
	return e, .94 + .06*e
}

// zoomAt 以 (px, py)(相对图片区域左上角)为中心缩放 f 倍:光标下的那一点在缩放前后保持不动,
// 与 Preview / 地图 / 设计工具的滚轮缩放一致。
func (lb *lightboxState) zoomAt(f, px, py float32) {
	old := lb.zoom
	nz := clamp(old*f, lbZoomMin, lbZoomMax)
	if math.Abs(float64(nz-1)) < 0.01 {
		nz = 1
	}
	if nz == old {
		return
	}
	r := nz / old
	cx, cy := px-lb.aw/2, py-lb.ah/2 // 图片中心偏移是相对区域中心记的
	lb.offX = cx - (cx-lb.offX)*r
	lb.offY = cy - (cy-lb.offY)*r
	lb.zoom = nz
}

func (a *App) lbRefreshNatural() {
	if a.lb == nil || a.lb.idx < 0 || a.lb.idx >= len(a.lb.paths) {
		return
	}
	p := a.lb.paths[a.lb.idx]
	a.lb.miss = false
	a.lb.loaded = false
	if _, err := os.Stat(p); err != nil {
		a.lb.miss = true
		a.lb.natW, a.lb.natH = 1200, 900
		a.lb.loaded = true
		return
	}
	if src := a.localImage(p); src != nil {
		iw, ih := ui.ImageSize(src)
		if iw > 0 && ih > 0 {
			a.lb.natW, a.lb.natH = iw, ih
			a.lb.loaded = true
			return
		}
	}
	a.lb.natW, a.lb.natH = 1200, 900
	a.lb.loaded = true
}

func (a *App) lightboxOverlay(c *ui.Context) {
	lb := a.lb
	if lb == nil {
		return
	}
	t := a.tk
	if !lb.closeAt.IsZero() && kit.Since(lb.closeAt) >= lbCloseDur {
		a.lb = nil
		return
	}
	alpha, scale := lb.lbProgress()
	if alpha < 1 || !lb.closeAt.IsZero() {
		a.frameSoon() // 动画期间持续出帧
	}
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(lbScrim).Grow(1).Opacity(alpha)
		closeFn := func() { a.closeLightbox() }
		backdrop.OnClick(func() {
			if lb.drag != nil && lb.drag.moved {
				return
			}
			closeFn()
		})
		panel.Grow(1).AlignSelf(ui.Stretch).Column() // 铺满整个窗口:此前面板只有内容那么宽(约 270),滚轮 / 拖动只在这一小块里有效
		panel.Children(func() {
			panel.OnShortcut(0, ui.KeyEscape, closeFn)
			panel.OnShortcut(0, ui.KeyEqual, func() { a.lbZoomBy(1.25) })
			panel.OnShortcut(ui.Shift, ui.KeyEqual, func() { a.lbZoomBy(1.25) })
			panel.OnShortcut(0, ui.KeyMinus, func() { a.lbZoomBy(1 / 1.25) })
			panel.OnShortcut(0, ui.Key0, func() { lb.zoom, lb.offX, lb.offY = 1, 0, 0 })
			if len(lb.paths) > 1 {
				panel.OnShortcut(0, ui.KeyLeft, func() { a.lbStep(-1) })
				panel.OnShortcut(0, ui.KeyRight, func() { a.lbStep(1) })
			}
			vw, vh := panel.Bounds().W, panel.Bounds().H
			if vw <= 0 || vh <= 0 {
				vw, vh = a.winW, a.winH
			}
			name := lb.names[lb.idx]
			if len(lb.paths) > 1 {
				name = name + " · " + itoa(lb.idx+1) + " / " + itoa(len(lb.paths))
			}
			ui.Row(c).Height(48).Padding(0, 16).AlignItems(ui.Center).Children(func() {
				ui.Text(c, name).FontSize(theme.FsBody).TextColor(lbInkA(.85)).Grow(1).SingleLine()
				a.lbTopBtn(c, "lb-x", "x", "关闭 (Esc)", closeFn)
			})
			area := ui.Box(c).Grow(1).MinHeight(0)
			area.HandleInput(lbPanZoomInput(lb))
			if lb.drag != nil && lb.drag.moved {
				area.Cursor(ui.CursorGrabbing)
			} else {
				area.Cursor(ui.CursorGrab)
			}
			if b := area.Bounds(); b.W > 0 && b.H > 0 {
				lb.aw, lb.ah = b.W, b.H
				vw, vh = b.W, b.H // 图片按图片区域居中 / 适应(此前按整个面板算,垂直方向偏了半个工具条)
			}
			dw, dh := lightboxDisplaySize(lb.natW, lb.natH, vw, vh, lb.zoom)
			if area.Pressed() {
				if dx, dy, ok := area.Dragged(); ok {
					// Dragged 返回的是「自上一帧起」指针挪动的量,不是从按下起的总量:要累加
					if lb.drag == nil {
						lb.drag = &lbDrag{}
					}
					d := lb.drag
					d.sx += dx
					d.sy += dy
					if !d.moved && (d.sx*d.sx+d.sy*d.sy) >= lbDragSlop*lbDragSlop {
						d.moved = true
						dx, dy = d.sx, d.sy // 越过阈值的那一帧把累积量一次补上,图片不会少走一截
					}
					if d.moved {
						lb.offX += dx
						lb.offY += dy
					}
				}
			} else if lb.drag != nil {
				lb.drag = nil
			}
			lx := (vw-dw)/2 + lb.offX
			ly := (vh-dh)/2 + lb.offY
			// 点图片外面的空白关闭,点图片本身(含拖动结束)不关
			area.OnClick(func() {
				if lb.drag != nil && lb.drag.moved {
					return
				}
				x, y, ok := area.PointerPosition()
				if ok && x >= lx && x <= lx+dw && y >= ly && y <= ly+dh {
					return
				}
				closeFn()
			})
			area.Draw(func(p *ui.Painter, r ui.Rect) {
				_ = p
				_ = r
			})
			area.Children(func() {
				if lb.miss {
					ui.Column(c).Absolute().Left(lx).Top(ly).Width(max(dw, 280)).Padding(24).Gap(12).
						Background(ui.Color{R: 0, G: 0, B: 0, A: 140}).Radius(t.RLg()).AlignItems(ui.Center).Children(func() {
						kit.Ic(c, "image", kit.S20).TextColor(lbInkA(.6))
						ui.Text(c, baseName(lb.paths[lb.idx])).FontSize(theme.FsBody).TextColor(lbInkA(.85))
						ui.Text(c, "找不到图片文件").FontSize(theme.FsCap).TextColor(lbInkA(.55))
						kit.Btn(c, "在访达中显示", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
							_ = exec.Command("open", "-R", lb.paths[lb.idx]).Start()
						})
					})
					return
				}
				src := a.localImage(lb.paths[lb.idx])
				// 图片自己不接任何指针事件(没有点击 / 光标):按下与拖动都落在外面的图片区域上,
				// 此前图片上挂了一个吞点击的空处理,按下被它拿走,图片上怎么拖都不动
				imgBox := ui.Box(c).Absolute().Left(lx).Top(ly).Size(dw, dh).Scale(scale, scale)
				imgBox.Children(func() {
					if src != nil {
						ui.Image(c, src).Size(dw, dh).Fit(ui.Contain)
					} else {
						ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Gap(8).Children(func() {
							kit.Spin(c, lbInkA(.7))
							ui.Text(c, "载入中…").FontSize(theme.FsCap).TextColor(lbInkA(.6))
						})
					}
				})
			})
			pct := lightboxZoomPct(lb.natW, lb.natH, vw, vh, lb.zoom)
			ui.Row(c).Padding(0, 0, 24, 0).Justify(ui.Center).Children(func() {
				bar := ui.Row(c).Gap(4).Padding(8, 12).Radius(t.RLg()).
					Background(ui.Color{R: 0, G: 0, B: 0, A: 160}).Border(1, lbInkA(.12)).AlignItems(ui.Center)
				bar.OnClick(func() {})
				bar.Children(func() {
					a.lbBarBtn(c, "lb-out", "minus", "缩小 (-)", func() { a.lbZoomBy(1 / 1.25) })
					ui.Text(c, itoa(pct)+"%").Width(52).TextAlign(ui.Center).FontSize(theme.FsCap).TextColor(lbInkA(.85))
					a.lbBarBtn(c, "lb-in", "plus", "放大 (+)", func() { a.lbZoomBy(1.25) })
					a.lbBarBtn(c, "lb-fit", "maximize", "适应窗口 (0)", func() { lb.zoom, lb.offX, lb.offY = 1, 0, 0 })
					ui.Box(c).Width(1).Height(20).Background(lbInkA(.15)).Margin(0, 4)
					a.lbBarBtn(c, "lb-open", "ext", "在预览中打开", func() {
						_ = exec.Command("open", lb.paths[lb.idx]).Start()
					})
					a.lbBarBtn(c, "lb-cp", "copy", "复制路径", func() {
						a.copyOut(lb.paths[lb.idx], "路径")
					})
				})
			})
		})
	})
	if !open {
		a.closeLightbox()
	}
}

func lbPanZoomInput(lb *lightboxState) func(ev ui.InputEvent) bool {
	return func(ev ui.InputEvent) bool {
		switch ev.Kind {
		case ui.InputZoom:
			if ev.Scale <= 0 {
				return false
			}
			lb.zoomAt(ev.Scale, ev.X, ev.Y) // 触控板捏合:以手指位置为中心
			return true
		case ui.InputScroll:
			if ev.Precise && ev.Mods&(ui.Ctrl|ui.Cmd) == 0 {
				// 触控板两指滑动 = 平移(与 Mermaid / 画布全屏同一套);鼠标滚轮才缩放
				lb.offX -= ev.DX
				lb.offY -= ev.DY
				return true
			}
			if ev.DY == 0 {
				return false
			}
			lb.zoomAt(lightboxWheelFactor(ev.DY), ev.X, ev.Y) // 鼠标滚轮(或 ⌘ + 滑动):以光标为中心
			return true
		}
		return false
	}
}

func (a *App) lbZoomBy(f float32) {
	if a.lb == nil {
		return
	}
	a.lb.zoomAt(f, a.lb.aw/2, a.lb.ah/2) // 键盘 / 按钮缩放:以图片区域中心为中心
}

func (a *App) lbStep(d int) {
	if a.lb == nil || len(a.lb.paths) < 2 {
		return
	}
	a.lb.idx = (a.lb.idx + d + len(a.lb.paths)) % len(a.lb.paths)
	a.lb.zoom, a.lb.offX, a.lb.offY = 1, 0, 0
	a.lbRefreshNatural()
}

func (a *App) lbTopBtn(c *ui.Context, key, icon, tip string, fn func()) {
	b := ui.ButtonBase(c).Key(key).Size(32, 32).Radius(a.tk.RSm()).Justify(ui.Center).AlignItems(ui.Center).Cursor(ui.CursorPointer)
	if b.Hovered() {
		b.Background(lbInkA(.16))
	}
	if b.Pressed() {
		b.Background(lbInkA(.28))
	}
	b.Tooltip(tip).OnClick(fn).Children(func() {
		kit.Ic(c, icon, kit.S16).TextColor(lbInk)
	})
}

func (a *App) lbBarBtn(c *ui.Context, key, icon, tip string, fn func()) {
	b := ui.ButtonBase(c).Key(key).Size(32, 32).Radius(a.tk.RSm()).Justify(ui.Center).AlignItems(ui.Center).Cursor(ui.CursorPointer)
	if b.Hovered() {
		b.Background(lbInkA(.16))
	}
	if b.Pressed() {
		b.Background(lbInkA(.28))
	}
	b.Tooltip(tip).OnClick(fn).Children(func() {
		kit.Ic(c, icon, kit.S16).TextColor(lbInk)
	})
}

// openUserImageLightbox 打开某条用户消息里的第 i 张图(同消息多图可左右切换)。
func (a *App) openUserImageLightbox(u *core.User, i int) {
	var paths, names []string
	for _, raw := range u.Images {
		abs := a.resolveUserImage(raw)
		if abs == "" {
			abs = raw
		}
		paths = append(paths, abs)
		names = append(names, baseName(raw))
	}
	if len(paths) == 0 {
		return
	}
	if i < 0 || i >= len(paths) {
		i = 0
	}
	a.openLightbox(paths, names, i)
}

func (a *App) openImageLightboxPath(abs string) {
	if strings.TrimSpace(abs) == "" {
		return
	}
	a.openLightbox([]string{abs}, []string{baseName(abs)}, 0)
}
