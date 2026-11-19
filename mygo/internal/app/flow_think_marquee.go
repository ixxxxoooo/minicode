package app

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 思考折叠 + 流式时,标题右侧的单行预览:只显示思考里「最新的一句」,新的一句开始时
// 旧句向上滚出、新句从下面翻进来(上下翻页),不再横向平移。

const (
	thinkLineMaxRunes = 160                    // 一句话最多取多少字(再长也只显示一行,超出省略)
	thinkMqMinW       = 96                     // 预览框比这还窄就不画
	thinkMqRollDur    = 260 * time.Millisecond // 上下翻一次的时长
	thinkMqLineH      = 20                     // 预览行高
)

type thinkMqSt struct {
	Line string    // 当前显示的句子
	Prev string    // 正在翻出去的上一句;空 = 没在翻
	At   time.Time // 这次翻页的开始时刻
}

// thinkLastLine 取思考正文里最新的一句:按换行 / 句末标点(。！？；与后面跟空格的 . ! ? ;)切分,
// 取最后一句;刚写完一句、末尾是标点时,取的仍是这一句(下一句出了第一个字才翻页)。
func thinkLastLine(text string) string {
	rs := []rune(strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n")))
	isTerm := func(i int) bool {
		switch rs[i] {
		case '\n', '。', '！', '？', '；', '!', '?', ';':
			return true
		case '.':
			return i+1 >= len(rs) || rs[i+1] == ' ' || rs[i+1] == '\n'
		}
		return false
	}
	end := len(rs)
	for end > 0 && (isTerm(end-1) || rs[end-1] == ' ' || rs[end-1] == '\t') {
		end--
	}
	start := end
	for start > 0 && !isTerm(start-1) {
		start--
	}
	line := strings.TrimSpace(string(rs[start:end]))
	line = strings.TrimLeft(line, "#*->` \t") // 去掉 Markdown 的标题 / 列表 / 引用 / 加粗记号
	if utf8.RuneCountInString(line) > thinkLineMaxRunes {
		line = string([]rune(line)[:thinkLineMaxRunes])
	}
	return line
}

func (a *App) thinkMarqueeVisible(sp step, open bool) bool {
	if !sp.thinkLive || open || !sp.hasBody || strings.TrimSpace(sp.thinkText) == "" {
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
	line := thinkLastLine(sp.thinkText)
	if line == "" {
		return ui.Box(c).Width(0).Shrink(0)
	}
	rowW := float32(0)
	if b := row.Bounds(); b.W > 0 {
		rowW = b.W
	}
	maxW := rowW * 0.52
	if maxW > 0 && maxW < thinkMqMinW {
		return ui.Box(c).Width(0).Shrink(0)
	}
	box := ui.Box(c).Height(thinkMqLineH).Grow(1).MinWidth(0).Shrink(1).Clip().MarginStart(4)
	box.MaxWidth(maxW)
	if b := box.Bounds(); b.W > 8 && b.W < thinkMqMinW {
		return ui.Box(c).Width(0).Shrink(0)
	}

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
			ui.Text(c, s).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine().Ellipsis("…").Shrink(1).MinWidth(0).PaddingStart(8)
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
