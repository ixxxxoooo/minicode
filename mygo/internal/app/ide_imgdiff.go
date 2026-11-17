package app

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// @author ixxxxoooo

// 差异标签里的图片 / 二进制文件:不能当文本逐行对比,图片改成「旧版 | 新版」并排预览(同一缩放比例,
// 可切到只看旧版或新版),新增 / 删除的图片只显示存在的一侧;其它二进制只给一行说明与大小变化。
// 内容在后台解码,界面线程只查结果;SVG 同时有文本,可以一键切回逐行文本对比。

// imgSide 是图片对比里的一侧(旧版或新版)。
type imgSide struct {
	has    bool // 这一侧有这个文件
	size   int64
	img    ui.ImageSource
	w, h   float32
	err    string
	format string // 「PNG · 512 × 512 · 38 KB」
}

// imgDiffState 是图片 / 二进制差异标签的状态。
type imgDiffState struct {
	kind            string // image / binary
	textOK          bool   // 有可用的文本对比(SVG),可切回逐行对比
	rawOld, rawNew  string // 两侧原始字节(Go 字符串能无损装二进制);变了就重新载入
	loading, loaded bool
	old, nw         imgSide
	view            int8    // 0 并排,1 只看旧版,2 只看新版
	zoom            float32 // 0 = 适应窗口
	bw, bh          float32 // 上一帧正文区尺寸(算适应比例用)
}

// diffMediaState 判断某个差异标签该不该走图片 / 二进制视图,返回初始状态(nil = 按文本对比)。
// 参数:p 文件路径;oldSrc / newSrc 两侧原始内容。
func diffMediaState(p, oldSrc, newSrc string) *imgDiffState {
	ext := path.Ext(p)
	switch {
	case isImageExt(ext):
		return &imgDiffState{kind: "image", rawOld: oldSrc, rawNew: newSrc, textOK: strings.EqualFold(ext, ".svg")}
	case looksBinary(oldSrc) || looksBinary(newSrc):
		return &imgDiffState{kind: "binary", rawOld: oldSrc, rawNew: newSrc}
	}
	return nil
}

// loadImgSide 解码一侧;raw 为空表示这一侧没有该文件。
func loadImgSide(raw, ext string, withImage bool) imgSide {
	s := imgSide{has: raw != "", size: int64(len(raw))}
	if !s.has {
		return s
	}
	if !withImage {
		s.format = humanBytes(s.size)
		return s
	}
	if s.size > viewImageMax {
		s.err = "图片太大(" + humanBytes(s.size) + "),不预览"
		return s
	}
	img, err := decodeAnyImage([]byte(raw), ext)
	if err != nil || img == nil {
		s.err = "图片无法解码(文件损坏或格式不受支持)"
		return s
	}
	s.img = img
	s.w, s.h = ui.ImageSize(img)
	s.format = fmt.Sprintf("%s · %d × %d · %s", strings.ToUpper(strings.TrimPrefix(ext, ".")), int(s.w+.5), int(s.h+.5), humanBytes(s.size))
	return s
}

// imgDiffLoad 第一次显示 / 内容变了之后在后台解码两侧。
func (a *App) imgDiffLoad(tb *ideTab) {
	v := tb.imgd
	if v.loading || v.loaded {
		return
	}
	v.loading = true
	rawOld, rawNew, ext, p := v.rawOld, v.rawNew, path.Ext(tb.path), tb.path
	withImg := v.kind == "image"
	a.bg("imgdiff-load", func() func() {
		end := diag.SpanSlow("viewer", "解码差异图片", 400*time.Millisecond, "path", p)
		o, n := loadImgSide(rawOld, ext, withImg), loadImgSide(rawNew, ext, withImg)
		end()
		viewLog.Info("差异图片载入完成", "path", p, "旧", o.has, "新", n.has)
		return func() {
			if v.rawOld != rawOld || v.rawNew != rawNew { // 载入期间内容又变了:丢弃,下一帧重来
				v.loading = false
				return
			}
			v.loading, v.loaded, v.old, v.nw = false, true, o, n
		}
	})
}

// refreshImgDiff 在仓库状态重读后同步图片差异标签的两侧内容;没变就什么都不做。
func (a *App) refreshImgDiff(tb *ideTab, oldSrc, newSrc string) {
	v := tb.imgd
	if v.rawOld == oldSrc && v.rawNew == newSrc {
		return
	}
	v.rawOld, v.rawNew, v.loaded = oldSrc, newSrc, false
}

// scale 算当前缩放比例:适应窗口时按两侧中最大的图、单个窗格大小,保证两侧同一比例。
func (v *imgDiffState) scale(panes int) float32 {
	if v.zoom > 0 {
		return v.zoom
	}
	mw, mh := max(v.old.w, v.nw.w), max(v.old.h, v.nw.h)
	if mw <= 0 || mh <= 0 || v.bw <= 0 || v.bh <= 0 {
		return 1
	}
	s := min((v.bw/float32(panes)-40)/mw, (v.bh-72)/mh, 1)
	return max(s, .02)
}

func (v *imgDiffState) zoomBy(f float32, panes int) {
	v.zoom = min(max(v.scale(panes)*f, .05), 16)
}

// imgDiffTab 是图片 / 二进制差异标签的内容。
func (a *App) imgDiffTab(c *ui.Context, tb *ideTab) {
	t, v := a.tk, tb.imgd
	a.imgDiffLoad(tb)
	a.scmTick() // 与文本差异一样:可见时定时重读仓库状态
	panes := 2
	if v.view != 0 {
		panes = 1
	}
	eff := v.scale(panes)
	ui.Row(c).Height(theme.Px(30)).Shrink(0).PaddingX(12).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
		a.fileIc(c, path.Base(tb.path))
		ui.Text(c, path.Base(tb.path)).FontWeight(500).SingleLine().Shrink(0)
		if !a.wsNarrow() {
			ui.Text(c, tb.diffLabel()).TextColor(t.Tx3).SingleLine()
		}
		if v.loaded {
			switch {
			case !v.old.has && v.nw.has:
				kit.Tag(c, "新增", kit.KOk)
			case v.old.has && !v.nw.has:
				kit.Tag(c, "已删除", kit.KErr)
			case v.rawOld == v.rawNew:
				kit.Tag(c, "内容相同", kit.KNeutral)
			default:
				kit.Tag(c, "已修改", kit.KAc)
			}
		}
		ui.Spacer(c).Grow(1)
		if v.kind == "image" && v.old.has && v.nw.has {
			if i := kit.Seg(c, []kit.SegItem{{Text: "并排"}, {Text: "旧版"}, {Text: "新版"}}, int(v.view), kit.SegOpt{Key: "imgdseg"}); i >= 0 {
				v.view = int8(i)
			}
		}
		if v.kind == "image" && v.loaded {
			kit.Ib(c, "minus", kit.IbOpt{Small: true, Tip: "缩小", Disabled: eff <= .05}).OnClick(func() { v.zoomBy(1/1.25, panes) })
			ui.Text(c, fmt.Sprintf("%d%%", int(eff*100+.5))).FontSize(theme.FsCap).TextColor(t.Tx3)
			kit.Ib(c, "plus", kit.IbOpt{Small: true, Tip: "放大", Disabled: eff >= 16}).OnClick(func() { v.zoomBy(1.25, panes) })
			if v.zoom == 0 {
				kit.Btn(c, "100%", kit.BtnOpt{Small: true, Tip: "按原始大小显示"}).OnClick(func() { v.zoom = 1 })
			} else {
				kit.Btn(c, "适应窗口", kit.BtnOpt{Small: true}).OnClick(func() { v.zoom = 0 })
			}
		}
		if v.textOK {
			kit.Btn(c, "文本对比", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "fileText"}).OnClick(func() { tb.imgd = nil })
		}
		if !a.wsNarrow() && v.nw.has {
			kit.Btn(c, "打开文件", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "ext"}).OnClick(func() { a.jumpTo(tb.path, 0, 0, 0) })
		}
	})
	body := ui.Column(c).Key("imgdbody-" + tb.id).Grow(1).MinHeight(0).MinWidth(0).Background(t.Fill1)
	if b := body.Bounds(); b.W > 0 {
		v.bw, v.bh = b.W, b.H
	} else {
		a.frameSoon()
	}
	body.Children(func() {
		switch {
		case !v.loaded:
			ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Gap(8).Children(func() {
				kit.Spin(c, t.Tx3)
				ui.Text(c, "正在载入…").FontSize(theme.FsCap).TextColor(t.Tx3)
			})
		case v.kind == "binary":
			a.binDiffBody(c, v)
		case v.view == 1:
			a.imgPane(c, tb, "旧版", &v.old, eff, "old")
		case v.view == 2:
			a.imgPane(c, tb, "新版", &v.nw, eff, "new")
		default:
			ui.Row(c).Grow(1).MinHeight(0).Children(func() {
				a.imgPane(c, tb, "旧版", &v.old, eff, "old")
				ui.Box(c).Width(1).Background(t.Bd1).Shrink(0)
				a.imgPane(c, tb, "新版", &v.nw, eff, "new")
			})
		}
	})
}

// imgPane 画一侧:标题行(旧版 / 新版 + 尺寸大小)+ 图片;这一侧没有文件时给说明。
func (a *App) imgPane(c *ui.Context, tb *ideTab, title string, s *imgSide, eff float32, key string) {
	t := a.tk
	ui.Column(c).Grow(1).Basis(0).MinWidth(0).MinHeight(0).Children(func() {
		ui.Row(c).Height(theme.Px(24)).Shrink(0).PaddingX(12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, title).FontSize(theme.FsCap).FontWeight(500).TextColor(t.Tx2).Shrink(0)
			if s.has && s.err == "" {
				ui.Text(c, s.format).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
			}
		})
		switch {
		case !s.has:
			a.imgPaneNote(c, map[string]string{"old": "旧版没有这个文件(新增)", "new": "新版没有这个文件(已删除)"}[key])
		case s.err != "":
			a.imgPaneNote(c, s.err)
		case tb.imgd.zoom == 0:
			ui.Column(c).Grow(1).MinHeight(0).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
				ui.Image(c, s.img).Size(s.w*eff, s.h*eff)
			})
		default:
			kit.VScroll(c).Key("imgdv-" + tb.id + key).Grow(1).MinHeight(0).Children(func() {
				kit.HScroll(c).Key("imgdh-" + tb.id + key).Children(func() {
					ui.Column(c).Padding(20).Children(func() {
						ui.Image(c, s.img).Size(s.w*eff, s.h*eff)
					})
				})
			})
		}
	})
}

// imgPaneNote 在窗格正中写一行灰字说明。
func (a *App) imgPaneNote(c *ui.Context, msg string) {
	ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
		ui.Text(c, msg).FontSize(theme.FsCap).TextColor(a.tk.Tx3)
	})
}

// binDiffBody 是其它二进制文件的差异:说明不能逐行对比,并列出两侧大小。
func (a *App) binDiffBody(c *ui.Context, v *imgDiffState) {
	t := a.tk
	ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Gap(8).Children(func() {
		kit.Ic(c, "file", kit.S20).TextColor(t.Tx4)
		ui.Text(c, "这是二进制文件,不能逐行对比").TextColor(t.Tx3)
		sz := func(s imgSide) string {
			if !s.has {
				return "无"
			}
			return humanBytes(s.size)
		}
		ui.Text(c, "旧版 "+sz(v.old)+"  →  新版 "+sz(v.nw)).FontSize(theme.FsCap).TextColor(t.Tx4)
	})
}
