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

	mu      sync.Mutex
	sink    backend.Sink
	stop    chan struct{}
	closed  bool
	gen     int                      // 播放代数:取消 / 新回合时加一,旧的播放协程见到变化就退出
	pending map[string]func() []Step // 等待用户处理的权限 / 提问 / 计划 → 处理后接着播放的剧情
	seq     int                      // 互动回合的条目编号
	Instant bool                     // 历史会话:Start 时同步一次性归约,不按节奏播放
	Auto    string                   // 启动后自动发出的提示(演示「权限请求」这类会话)
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
	if b.Instant {
		for _, st := range b.script() {
			sink(st.Ev)
		}
	} else {
		b.play(b.script())
	}
	if b.Auto != "" {
		b.prompt(core.CmdPrompt{Text: b.Auto})
	}
}

// play 异步按节奏播放一段剧情;新的 play 或 Cancel 会让旧的播放停下。
func (b *Backend) play(steps []Step) {
	b.mu.Lock()
	b.gen++
	gen, sink := b.gen, b.sink
	b.mu.Unlock()
	if sink == nil {
		return
	}
	go func() {
		defer diag.Recover("demo.play")
		started := time.Now()
		defer func() { diag.L("demo").Debug("剧本播放结束", "steps", len(steps), "耗时", time.Since(started)) }()
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
			b.mu.Lock()
			alive := b.gen == gen
			b.mu.Unlock()
			if !alive {
				return
			}
			sink(st.Ev)
		}
	}()
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
