package app

import (
	"math"
)

// 图片灯箱的缩放范围(相对「适应窗口」的倍数)。
const (
	lbZoomMin   = 0.2
	lbZoomMax   = 12
	lbDragSlop  = 3
	lbFitW      = 0.9
	lbFitH      = 0.82
	lbWheelUnit = 40
)

// lightboxFitScale 计算适应窗口的缩放系数(不放大超过原图)。
func lightboxFitScale(nw, nh, vw, vh float32) float32 {
	if nw <= 0 || nh <= 0 || vw <= 0 || vh <= 0 {
		return 1
	}
	k := (vw * lbFitW / nw)
	if k2 := vh * lbFitH / nh; k2 < k {
		k = k2
	}
	if k > 1 {
		k = 1
	}
	return k
}

// lightboxDisplaySize 返回适应 + 用户缩放后的显示宽高。
func lightboxDisplaySize(nw, nh, vw, vh, zoom float32) (w, h float32) {
	fit := lightboxFitScale(nw, nh, vw, vh)
	z := clamp(zoom, lbZoomMin, lbZoomMax)
	return nw * fit * z, nh * fit * z
}

// lightboxZoomPct 返回工具条显示用的百分比(相对原图像素)。
func lightboxZoomPct(nw, nh, vw, vh, zoom float32) int {
	fit := lightboxFitScale(nw, nh, vw, vh)
	return int(math.Round(float64(fit * zoom * 100)))
}

// lightboxWheelFactor 滚轮增量 → 缩放倍数。
func lightboxWheelFactor(dy float32) float32 {
	f := math.Pow(1.1, float64(-dy)/lbWheelUnit)
	return float32(math.Min(2, math.Max(0.5, f)))
}

// pinnedEditorChrome 悬浮编辑卡正文区之外的固定高度(内边距、按钮行、提示、边框)。
const pinnedEditorChrome = 100

// pinnedEditorMinBody 悬浮编辑卡正文区最小高度。
const pinnedEditorMinBody = 60

// pinnedEditorInset 吸附卡与编辑卡相对会话流顶边的间距。
const pinnedEditorInset = 8

// pinnedEditorBodyMax 正文区最大高度:视口剩余高度扣掉顶边、底边间距与固定部分。
func pinnedEditorBodyMax(viewportH, top float32) float32 {
	return max(viewportH-top-pinnedEditorInset*2-pinnedEditorChrome, pinnedEditorMinBody)
}
