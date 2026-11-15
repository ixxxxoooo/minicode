package demo

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// stepBuf 瞬时回放用的事件缓冲(After 恒为 0)。
type stepBuf struct {
	steps []Step
}

func (b *stepBuf) cap(n int) {
	if cap(b.steps) < n {
		b.steps = make([]Step, 0, n)
	}
}

func (b *stepBuf) ev(ev core.Event) {
	b.steps = append(b.steps, Step{0, ev})
}

func (b *stepBuf) out() []Step { return b.steps }

// detRand 确定性伪随机(与 math/rand 无关,跨平台一致)。
type detRand struct{ s uint64 }

func newDetRand(seed uint64) *detRand { return &detRand{s: seed} }

func (r *detRand) next() uint64 {
	r.s = r.s*6364136223846793005 + 1442695040888963407
	return r.s
}

func (r *detRand) intn(n int) int {
	if n <= 0 {
		return 0
	}
	return int(r.next() % uint64(n))
}

func (r *detRand) pick(ss ...string) string {
	if len(ss) == 0 {
		return ""
	}
	return ss[r.intn(len(ss))]
}

// genLines 生成 n 行文本,每行前缀可定制(用于命令输出 / diff)。
func genLines(n int, prefix string) string {
	if n <= 0 {
		return ""
	}
	avg := len(prefix) + 12
	var b strings.Builder
	b.Grow(n * avg)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "%s%d\n", prefix, i)
	}
	return b.String()
}

// genANSILines 带 ANSI 色与一条超长行的终端输出。
func genANSILines(n int) string {
	if n <= 0 {
		return ""
	}
	var b strings.Builder
	b.Grow(n * 48)
	palette := []string{"\x1b[32m", "\x1b[33m", "\x1b[36m", "\x1b[35m", "\x1b[0m"}
	for i := 1; i <= n; i++ {
		c := palette[i%len(palette)]
		if i == n/2 {
			long := strings.Repeat("WARN=", 400)
			fmt.Fprintf(&b, "%s[%06d] %s trace overflow\x1b[0m\n", c, i, long)
			continue
		}
		fmt.Fprintf(&b, "%s[%06d] ok line detail …\x1b[0m\n", c, i)
	}
	return b.String()
}

// genDiffBody 生成约 lines 行的单文件 diff 正文(交替增删)。
func genDiffBody(lines int, path string) (old, new string) {
	var o, n strings.Builder
	o.Grow(lines * 24)
	n.Grow(lines * 24)
	for i := 1; i <= lines; i++ {
		if i%3 == 0 {
			fmt.Fprintf(&o, "// %s:%d stale\n", path, i)
			fmt.Fprintf(&n, "// %s:%d fresh\n", path, i)
		} else if i%3 == 1 {
			fmt.Fprintf(&o, "let v%d = %d;\n", i, i)
			fmt.Fprintf(&n, "let v%d = %d;\n", i, i+1)
		} else {
			line := fmt.Sprintf("fn step_%d() {}\n", i)
			o.WriteString(line)
			n.WriteString(line)
		}
	}
	return o.String(), n.String()
}

// genMarkdownBulk 生成约目标字符数的 Markdown(标题 / 表格 / 列表 / 代码块循环)。
func genMarkdownBulk(targetRunes int, tag string) string {
	if targetRunes <= 0 {
		return ""
	}
	var b strings.Builder
	b.Grow(targetRunes * 3)
	section := 0
	for b.Len() < targetRunes {
		section++
		fmt.Fprintf(&b, "## %s · 小节 %d\n\n", tag, section)
		b.WriteString("说明：中英混排 — stress test for layout, **bold**, `inline`, 以及列表。\n\n")
		b.WriteString("| 列 A | 列 B | 列 C |\n| --- | --- | --- |\n")
		for row := 0; row < 4; row++ {
			fmt.Fprintf(&b, "| r%d | 数据 %d | ok |\n", row, section*10+row)
		}
		b.WriteString("\n- bullet one\n- bullet two\n- bullet 三\n\n")
		fmt.Fprintf(&b, "```go\nfunc Chunk%d() {{\n  // layer %d\n}}\n```\n\n", section, section)
	}
	rs := []rune(b.String())
	if len(rs) > targetRunes {
		rs = rs[:targetRunes]
	}
	return string(rs)
}

// genRunOn 生成单行超长文本(无换行)。
func genRunOn(chars int) string {
	if chars <= 0 {
		return ""
	}
	unit := "0123456789abcdef中英混排αβγ"
	reps := (chars / utf8.RuneCountInString(unit)) + 1
	var b strings.Builder
	b.Grow(chars + 8)
	for b.Len() < chars {
		b.WriteString(unit)
	}
	rs := []rune(b.String())
	return string(rs[:chars])
}

func (b *stepBuf) think(id, text string, secs uint64) {
	b.ev(core.EvAppend{Item: &core.Thinking{ID: id, Text: text}})
	b.ev(core.EvDuration{ID: id, Secs: secs})
}

func (b *stepBuf) say(id, text string) {
	b.ev(core.EvAppend{Item: &core.Assistant{ID: id, Text: text}})
}

func (b *stepBuf) toolDone(tc *core.ToolCall, dur int64, content ...core.ToolContent) {
	b.ev(core.EvAppend{Item: tc})
	patch := core.ToolPatch{Status: core.Status(core.ToolCompleted), DurationMs: dur}
	if len(content) > 0 {
		patch.AppendContent = content
	}
	b.ev(core.EvToolUpdate{ID: tc.ID, Patch: patch})
}

func (b *stepBuf) toolFail(tc *core.ToolCall, dur int64, msg string) {
	b.ev(core.EvAppend{Item: tc})
	b.ev(core.EvToolUpdate{ID: tc.ID, Patch: core.ToolPatch{
		Status: core.Status(core.ToolFailed), DurationMs: dur,
		AppendContent: []core.ToolContent{&core.TextBlock{Text: msg}},
	}})
}

func (b *stepBuf) endTurn(uid string, round int, reason core.StopReason) {
	stats := DefaultStats(round)
	stats.Input += uint64(round * 120)
	stats.Output += uint64(round % 17 * 40)
	secs := uint64(5 + round/12 + round%50)
	b.ev(core.EvTurnStats{ID: uid, Stats: *stats})
	b.ev(core.EvDuration{ID: uid, Secs: secs})
	b.ev(core.EvTurnEnded{Reason: reason})
}
