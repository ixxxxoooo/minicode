package app

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/core"
)

// 差异块的行级差异按指针缓存:同一块第二次取到同一份结果,内容变了就重算。
func TestBlockDiffCache(t *testing.T) {
	d := &core.DiffBlock{Path: "a.go", Old: "a\nb\n", New: "a\nc\nd\n", Line: 1}
	e1 := blockDiffOf(d)
	e2 := blockDiffOf(d)
	if e1 != e2 {
		t.Fatal("同一个块第二次应命中缓存")
	}
	if e1.add != 2 || e1.del != 1 {
		t.Fatalf("增删行数不对:+%d −%d", e1.add, e1.del)
	}
	ad, de := reviewStat(d)
	if ad != e1.add || de != e1.del {
		t.Fatal("reviewStat 应与缓存一致")
	}
	d.New = "a\nb\n"
	if e3 := blockDiffOf(d); e3 == e1 || e3.add != 0 || e3.del != 0 {
		t.Fatalf("块内容变了应重算:%+v", e3)
	}
}

// 大块差异的缓存应让第二次取值近乎零成本。
func TestBlockDiffCacheFast(t *testing.T) {
	var ob, nb strings.Builder
	for i := 0; i < 1500; i++ {
		ob.WriteString("line old " + itoa(i) + "\n")
		nb.WriteString("line new " + itoa(i*7%1500) + "\n")
	}
	d := &core.DiffBlock{Path: "big.txt", Old: ob.String(), New: nb.String(), Line: 1}
	t0 := time.Now()
	blockDiffOf(d)
	first := time.Since(t0)
	t1 := time.Now()
	for i := 0; i < 1000; i++ {
		blockDiffOf(d)
	}
	hit := time.Since(t1) / 1000
	t.Logf("首次=%v 命中=%v", first, hit)
	if hit > first/5 && hit > 50*time.Microsecond {
		t.Fatalf("命中缓存仍然很慢:首次=%v 命中=%v", first, hit)
	}
}

// dirty():文本与磁盘一致 → false;改了 → true;改回 → false;换磁盘文本 → 重新比较。
func TestDirtyMemo(t *testing.T) {
	src := strings.Repeat("hello world\n", 2000)
	tb := &ideTab{buf: editor.NewBuffer(src), saved: src}
	if tb.dirty() {
		t.Fatal("刚打开不该未保存")
	}
	if tb.dirty() { // 命中缓存
		t.Fatal("命中缓存后仍应为 false")
	}
	if !tb.dirtyM.ok {
		t.Fatal("同长度比较后应记下缓存")
	}
	// 同长度改动(替换一个字符):走完整比较
	tb.buf.Edit(editor.Edit{Start: 0, End: 1, Text: "H"})
	if !tb.dirty() {
		t.Fatal("同长度改动应为未保存")
	}
	tb.buf.Edit(editor.Edit{Start: 0, End: 1, Text: "h"})
	if tb.dirty() {
		t.Fatal("改回原文应不再未保存")
	}
	// 磁盘文本换了(保存):按新指针重新比较
	tb.buf.Edit(editor.Edit{Start: 0, End: 1, Text: "X"})
	tb.saved = tb.buf.String()
	if tb.dirty() {
		t.Fatal("保存后不该未保存")
	}
	// 长度不同:O(1) 快路径
	tb.buf.Edit(editor.Edit{Start: 0, End: 0, Text: "abc"})
	if !tb.dirty() {
		t.Fatal("长度变了应为未保存")
	}
	// 行尾改动也算未保存
	tb.buf.Edit(editor.Edit{Start: 0, End: 3, Text: ""})
	tb.eol = eolCRLF
	if !tb.dirty() {
		t.Fatal("改行尾应为未保存")
	}
}

// 大文件上 dirty() 命中缓存时不应再拼整篇字符串。
func BenchmarkDirtyBigFile(b *testing.B) {
	src := strings.Repeat("some line of source code here\n", 200000)
	tb := &ideTab{buf: editor.NewBuffer(src), saved: src}
	tb.dirty()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tb.dirty()
	}
}

// 探针:慢帧之后的追帧、画在舞台之外的底部面板,不应记成「跳变」。
func TestProbeCatchUpNotJump(t *testing.T) {
	a, _ := newTestApp(t)
	p := &a.pr
	p.suspend, p.busy = 0, false
	p.prev, p.frames, p.slowLeft, p.prevD = nil, nil, 0, [pzN]float32{}
	mk := func(dt time.Duration, chatW float32) *AnimFrame {
		f := &AnimFrame{Dt: dt, Win: 2560}
		f.Has[pzChat] = true
		f.R[pzChat].W = chatW
		return f
	}
	// 宽度序列(逐帧 -68,-200,-91):第二、三帧的二阶差分分别是 -132、+109。
	widths := []float32{1400, 1325, 1257, 1057, 966}
	dts := []time.Duration{0, 17 * time.Millisecond, 38 * time.Millisecond, 17 * time.Millisecond, 17 * time.Millisecond}
	var prev *AnimFrame
	var flagged int
	for i, w := range widths {
		f := mk(dts[i], w)
		p.prev = prev
		a.probeCheck(f)
		p.frames = append(p.frames, *f) // 与 probeEnd 一致:检查之后才入列
		if strings.Contains(f.Flags, "跳变") {
			flagged++
		}
		prev = f
	}
	if flagged != 0 {
		t.Fatalf("慢帧后的追帧不应记成跳变,实际 %d 帧", flagged)
	}
	// 真正的瞬移:一帧走完全程(前后帧位移接近 0)仍要报。
	p.prev, p.frames, p.slowLeft, p.prevD = nil, nil, 0, [pzN]float32{}
	prev = nil
	jump := 0
	for i, w := range []float32{800, 800, 800, 200, 200} {
		f := mk(17*time.Millisecond, w)
		p.prev = prev
		a.probeCheck(f)
		p.frames = append(p.frames, *f) // 与 probeEnd 一致:检查之后才入列
		if strings.Contains(f.Flags, "跳变") {
			jump++
		}
		prev = f
		_ = i
	}
	if jump == 0 {
		t.Fatal("一帧瞬移应当仍被记成跳变")
	}
}
