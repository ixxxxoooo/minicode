package app

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 思考折叠 + 流式时,「思考」后面不再放静态标题,而是一行上下翻的预览:只显示思考里「最新的一句」,
// 新的一句开始时旧句向上滚出、新句从下面翻进来,上下边缘渐隐;只按宽度省略,不按字数截断。
// 思考结束后行内是静态标题(第一句,同样只按宽度省略)。

const (
	thinkLineMaxRunes = 300                    // 一句话最多取多少字(只是防极端长句;显示上只按宽度省略,不按字数截断)
	thinkMqRollDur    = 260 * time.Millisecond // 上下翻一次的时长
	thinkMqLineH      = 20                     // 预览行高
)

type thinkMqSt struct {
	Line string    // 当前显示的句子
	Prev string    // 正在翻出去的上一句;空 = 没在翻
	At   time.Time // 这次翻页的开始时刻
}

// thinkTermAt 第 i 个字是不是句末:换行 / 。！？； / ! ? ;,以及后面跟空格(或在末尾)的英文句点。
func thinkTermAt(rs []rune, i int) bool {
	switch rs[i] {
	case '\n', '。', '！', '？', '；', '!', '?', ';':
		return true
	case '.':
		return i+1 >= len(rs) || rs[i+1] == ' ' || rs[i+1] == '\n'
	}
	return false
}

// thinkClean 去掉一句话两端的空白与开头的 Markdown 记号(标题 # / 列表 - / 引用 > / 加粗 * / 反引号),并限制长度。
func thinkClean(rs []rune) string {
	line := strings.TrimLeft(strings.TrimSpace(string(rs)), "#*->` \t")
	if utf8.RuneCountInString(line) > thinkLineMaxRunes {
		line = string([]rune(line)[:thinkLineMaxRunes])
	}
	return line
}

// thinkLastLine 取思考正文里最新的一句:按换行 / 句末标点切分,取最后一句;
// 刚写完一句、末尾是标点时,取的仍是这一句(下一句出了第一个字才翻页)。
func thinkLastLine(text string) string {
	rs := []rune(strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n")))
	end := len(rs)
	for end > 0 && (thinkTermAt(rs, end-1) || rs[end-1] == ' ' || rs[end-1] == '\t') {
		end--
	}
	start := end
	for start > 0 && !thinkTermAt(rs, start-1) {
		start--
	}
	return thinkClean(rs[start:end])
}

// thinkFirstLine 取思考正文的第一句(折叠行的标题);第一句还没写完时就是已有的全部文字。
func thinkFirstLine(text string) string {
	rs := []rune(strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n")))
	start := 0
	for start < len(rs) && (thinkTermAt(rs, start) || rs[start] == ' ' || rs[start] == '\t') {
		start++ // 跳过开头的空行 / 标点
	}
	end := start
	for end < len(rs) && !thinkTermAt(rs, end) {
		end++
	}
	return thinkClean(rs[start:end])
}

func (a *App) thinkMarqueeVisible(sp step, open bool) bool {
	if !sp.thinkLive || open || !sp.hasBody || thinkLastLine(sp.thinkText) == "" {
		return false
	}
	if a.thinkMq == nil {
		a.thinkMq = map[string]thinkMqSt{}
	}
	if a.chatTier() == tierNarrow {
		return false
	}
	if at, hide := a.thinkMqHide[sp.id]; hide && a.now().Sub(at) > 320*time.Millisecond {
		return false
	}
	return true
}

func (a *App) thinkMarqueeRow(c *ui.Context, sp step, tone ui.Color, open bool, row ui.Element) ui.Element {
	if !a.thinkMarqueeVisible(sp, open) {
		return ui.Box(c).Width(0).Shrink(0)
	}
	t := a.tk
	line := thinkLastLine(sp.thinkText) // 可见条件已保证非空
	// 预览顶替标题的位置(流式折叠时标题不再单独显示):占满「思考」与用时之间的整段宽度,
	// 上下边缘渐隐——翻页时滚出 / 滚入的句子在边缘淡出淡入,稳态的一行在中间、不受影响。
	box := ui.Box(c).Height(thinkMqLineH).Grow(1).MinWidth(0).Shrink(1).Clip()
	black := ui.RGB(0, 0, 0)
	box.MaskLinearGradient(ui.LinearGradient{Angle: 180, Stops: []ui.GradientStop{
		{Color: ui.Transparent, At: 0}, {Color: black, At: .2}, {Color: black, At: .8}, {Color: ui.Transparent, At: 1},
	}})

	still := !kit.MotionOn() || kit.SystemReduceMotion() || (a.win != nil && !a.win.IsFocused())
	id := sp.id
	st := a.thinkMq[id]
	now := a.now()
	if line != st.Line {
		// 还是同一句、只是又长了几个字:原地接着写,不翻页;换了一句才翻
		grow := st.Line != "" && strings.HasPrefix(line, st.Line)
		switch {
		case grow:
		case st.Line != "" && !still:
			st.Prev, st.At = st.Line, now
		default:
			st.Prev = ""
		}
		st.Line = line
	}
	p := float32(1) // 翻页进度 0→1(缓出)
	if st.Prev != "" {
		if d := now.Sub(st.At); d < thinkMqRollDur {
			x := 1 - float32(d)/float32(thinkMqRollDur)
			p = 1 - x*x*x
			a.frameSoon()
		} else {
			st.Prev = ""
		}
	}
	a.thinkMq[id] = st

	one := func(s string, alpha float32) {
		r := ui.Row(c).Height(thinkMqLineH).Shrink(0).MinWidth(0).AlignItems(ui.Center)
		if alpha < 1 {
			r.Opacity(alpha)
		}
		r.Children(func() {
			ui.Text(c, s).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
		})
	}
	box.Children(func() {
		if st.Prev == "" {
			one(st.Line, 1)
			return
		}
		// 两句上下叠放,整体向上平移一行:旧句滚出、新句从下面翻进来
		ui.Column(c).Height(2*thinkMqLineH).Shrink(0).Grow(1).MinWidth(0).Translate(0, -thinkMqLineH*p).Children(func() {
			one(st.Prev, 1-p)
			one(st.Line, p)
		})
	})
	return box
}

func (a *App) noteThinkStreamEnd(id string) {
	if a.thinkWasLive[id] {
		a.thinkMqHide[id] = a.now()
	}
	delete(a.thinkWasLive, id)
}
