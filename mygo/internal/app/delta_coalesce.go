package app

import (
	"sync"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// deltaWindow 是流式文字增量的合并窗口:窗口内到达的同一条目的增量拼成一个事件再归约,
// 每次归约都会触发一次界面重建与排版,模型一秒吐几十个词时,合并后每秒只重建约 25 次。
const deltaWindow = 40 * time.Millisecond

// deltaCoalescer 合并连续的 TextDelta / ThinkingDelta(同 Parent + ID + 同种类)。
// 保证事件顺序不变:任何其它事件到达前,先把攒着的增量放出去。
// 首个增量(距上次放出超过窗口)立即放出,不增加首字延迟。
type deltaCoalescer struct {
	mu      sync.Mutex
	pending core.Event // 攒着的增量,nil = 没有
	last    time.Time  // 上一次放出增量的时间
	timer   *time.Timer
	emit    func(core.Event) func() // 把事件包成「投递到主线程」的函数
}

// push 收一个事件:增量可能被攒起来,其它事件先冲掉攒着的增量再投递。
// now 是当前事件的投递函数,wrap 把攒着的合并事件包成投递函数。
func (d *deltaCoalescer) push(ev core.Event, now func(), wrap func(core.Event) func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.emit = wrap
	if !isDelta(ev) {
		d.flushLocked()
		now()
		return
	}
	if d.pending != nil {
		if merged, ok := mergeDelta(d.pending, ev); ok {
			d.pending = merged
			return
		}
		d.flushLocked() // 换了条目:先放出旧的
	}
	if time.Since(d.last) >= deltaWindow {
		d.last = time.Now()
		now()
		return
	}
	d.pending = ev
	if d.timer == nil {
		d.timer = time.AfterFunc(deltaWindow, func() {
			d.mu.Lock()
			defer d.mu.Unlock()
			d.flushLocked()
		})
	} else {
		d.timer.Reset(deltaWindow)
	}
}

// flushLocked 放出攒着的增量(调用方持有锁)。
func (d *deltaCoalescer) flushLocked() {
	if d.pending == nil {
		return
	}
	ev := d.pending
	d.pending = nil
	d.last = time.Now()
	if d.timer != nil {
		d.timer.Stop()
	}
	if d.emit != nil {
		d.emit(ev)()
	}
}

func isDelta(ev core.Event) bool {
	switch ev.(type) {
	case core.EvTextDelta, core.EvThinkingDelta:
		return true
	}
	return false
}

// mergeDelta 把 b 拼到 a 后面(同种类、同条目才行)。
func mergeDelta(a, b core.Event) (core.Event, bool) {
	switch x := a.(type) {
	case core.EvTextDelta:
		if y, ok := b.(core.EvTextDelta); ok && x.Parent == y.Parent && x.ID == y.ID {
			x.Delta += y.Delta
			return x, true
		}
	case core.EvThinkingDelta:
		if y, ok := b.(core.EvThinkingDelta); ok && x.Parent == y.Parent && x.ID == y.ID {
			x.Delta += y.Delta
			return x, true
		}
	}
	return nil, false
}
