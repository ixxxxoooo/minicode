package kit

import (
	"math"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

const (
	shimmerCycle = 2100 * time.Millisecond
	shimmerSweep = 1600 * time.Millisecond
	shimmerTick  = 33 * time.Millisecond
	shimmerBins  = 6
	shimmerMax   = 48
)

var viewActive atomic.Bool

// SetViewActive 窗口在前台且可见时才跑流光(失焦静止)。
func SetViewActive(on bool) { viewActive.Store(on) }

func viewOK() bool {
	return MotionOn() && viewActive.Load() && !SystemReduceMotion()
}

// shimmerBandPos 0..1,扫过后在终点停顿。
func shimmerBandPos() float32 {
	t := time.Now().UnixNano() % int64(shimmerCycle)
	if t >= int64(shimmerSweep) {
		return 1
	}
	return float32(t) / float32(shimmerSweep)
}

// RunTextS 为进行中标签上色(流光 / 呼吸);流光用 RichText 分段,限频约 30fps。
func RunTextS(c *ui.Context, text string, hi ui.Color, style func(ui.Element) ui.Element) ui.Element {
	switch shimmerMode {
	case 2:
		e := ui.Text(c, text)
		if style != nil {
			e = style(e)
		}
		return e
	case 1:
		e := ui.Text(c, text)
		if style != nil {
			e = style(e)
		}
		return Pulse(e)
	default:
		if !viewOK() {
			e := ui.Text(c, text).TextColor(hi.Mix(T.Tx3, .35))
			if style != nil {
				e = style(e)
			}
			return e
		}
		n := utf8.RuneCountInString(text)
		if n > shimmerMax {
			e := ui.Text(c, text)
			if style != nil {
				e = style(e)
			}
			return Pulse(e)
		}
		dim := T.Tx3
		pos := shimmerBandPos()
		spans := shimmerSpans(text, hi, dim, pos)
		rt := ui.RichText(c, spans...)
		if style != nil {
			rt = style(rt)
		}
		_ = rt.Loop("shimmer", shimmerTick, ui.Linear)
		return rt
	}
}

func shimmerSpans(text string, hi, dim ui.Color, pos float32) []ui.Span {
	runes := []rune(text)
	n := len(runes)
	if n == 0 {
		return []ui.Span{{Text: text, Color: dim}}
	}
	bins := min(shimmerBins, n)
	span := make([]ui.Span, 0, bins)
	per := float32(n) / float32(bins)
	center := pos * float32(n)
	band := float32(n) * .38
	for b := 0; b < bins; b++ {
		i0 := int(float32(b) * per)
		i1 := n
		if b < bins-1 {
			i1 = int(float32(b+1) * per)
		}
		mid := (float32(i0) + float32(i1)) / 2
		d := math.Abs(float64(mid - center))
		w := float64(d)/float64(band)*math.Pi - math.Pi/2
		bright := .35 + .65*(.5+.5*math.Sin(w))
		if w < -math.Pi/2 {
			bright = .35
		}
		col := dim.Mix(hi, float32(bright))
		span = append(span, ui.Span{Text: string(runes[i0:i1]), Color: col})
	}
	return span
}
