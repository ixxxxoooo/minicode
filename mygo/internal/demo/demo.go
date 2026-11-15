// Package demo 是脚本化的演示后端:不起任何子进程,按剧本把事件送给界面,
// 演示与真实后端共用同一套界面与归约器。
package demo

import (
	"sync"
	"time"

	"github.com/ixxxxoooo/minicode/internal/backend"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diag"
)

var log = diag.L("demo")

// Backend 按剧本播放事件。Speed 是慢放倍数(MINI_DEMO_SLOW,默认 1)。
type Backend struct {
	script func() []Step
	speed  float64

	mu     sync.Mutex
	sink   backend.Sink
	stop   chan struct{}
	closed bool
}

// New 新建演示后端。script 在 Start 时调用一次。
func New(script func() []Step, speed float64) *Backend {
	if speed <= 0 {
		speed = 1
	}
	return &Backend{script: script, speed: speed, stop: make(chan struct{})}
}

func (b *Backend) Name() string { return "demo" }

func (b *Backend) Start(sink backend.Sink) {
	b.mu.Lock()
	b.sink = sink
	b.mu.Unlock()
	steps := b.script()
	go func() {
		defer diag.Recover("demo.play")
		end := diag.Span("demo", "播放剧本", "steps", len(steps))
		defer end()
		for _, st := range steps {
			if st.After > 0 {
				t := time.NewTimer(time.Duration(float64(st.After) * b.speed))
				select {
				case <-t.C:
				case <-b.stop:
					t.Stop()
					return
				}
			}
			sink(st.Ev)
		}
	}()
}

func (b *Backend) Send(cmd core.Command) {
	log.Debug("命令", "cmd", cmd)
}

func (b *Backend) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.closed {
		b.closed = true
		close(b.stop)
	}
}

// Replay 把剧本一次性归约进 st(不等待、不起 goroutine),用于静态截图与测试。
func Replay(st *core.State, steps []Step) {
	for _, s := range steps {
		st.Apply(s.Ev)
	}
}
