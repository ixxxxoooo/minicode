package app

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// 布局动画回归测试:用 Tester 的确定性时钟逐帧推进(16 ms 一帧),
// 读探针(anim_probe.go)记下的各区宽度曲线,断言「无空白 / 无跳变 / 起止宽度正确 / 全程单调」。
// 环境变量 ANIM_DUMP=1 时把每个场景的逐帧表打到标准输出(go test -run Anim -v),用来人工看曲线。

// animRun 在确定性时钟下执行 act,然后推进 dur,返回探针记录的逐帧数据。
func animRun(t *testing.T, a *App, tt *ui.Tester, name string, act func(), dur time.Duration) []AnimFrame {
	t.Helper()
	kit.SetClock(tt.Now)
	a.driveFrames = true
	t.Cleanup(func() { kit.SetClock(nil); a.driveFrames = false })
	// 先让之前的动画与探针都静止
	tt.Advance(1200 * time.Millisecond)
	tt.Frame()
	a.AnimProbeKeep(true)
	act()
	tt.Frame()
	for el := time.Duration(0); el < dur; el += 16 * time.Millisecond {
		tt.Advance(16 * time.Millisecond)
	}
	tt.Advance(400 * time.Millisecond) // 让探针结算
	fr := a.AnimFrames()
	switch os.Getenv("ANIM_DUMP") {
	case "2":
		dumpAnim(name, fr)
	case "1":
		sumAnim(name, fr)
	}
	return fr
}

// sumAnim 打一行汇总:帧数 / 异常数 / 最大空白 / 各区最大单帧位移 / 起止宽度。
func sumAnim(name string, fr []AnimFrame) {
	if len(fr) == 0 {
		fmt.Printf("%-22s 无动画帧\n", name)
		return
	}
	var blank float32
	bad := 0
	var step [pzN]float32
	for i, f := range fr {
		blank = max(blank, f.Blank)
		if f.Flags != "" {
			bad++
		}
		if i > 0 {
			for z := 0; z < pzN; z++ {
				if f.Has[z] && fr[i-1].Has[z] {
					d := f.R[z].W - fr[i-1].R[z].W
					if z == pzPanel {
						d = f.R[z].H - fr[i-1].R[z].H
					}
					step[z] = max(step[z], absF(d))
				}
			}
		}
	}
	a, b := fr[0], fr[len(fr)-1]
	fmt.Printf("%-22s %2d帧 %4dms 异常%2d 最大空白%6.1f | 单帧位移 侧%5.1f 对话%5.1f 工作%5.1f 面板%5.1f | 侧%.0f→%.0f 对话%.0f→%.0f 工作%.0f→%.0f 面板H%.0f→%.0f\n",
		name, len(fr), b.T.Milliseconds(), bad, blank, step[pzSide], step[pzChat], step[pzWs], step[pzPanel],
		wOf(a, pzSide), wOf(b, pzSide), wOf(a, pzChat), wOf(b, pzChat), wOf(a, pzWs), wOf(b, pzWs), a.R[pzPanel].H, b.R[pzPanel].H)
}

func dumpAnim(name string, fr []AnimFrame) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "\n=== %s(%d 帧)===\n", name, len(fr))
	for _, f := range fr {
		fmt.Fprintf(&sb, "%4dms dt%2d  侧%6.1f  舞台%7.1f+%-6.1f  对话%7.1f+%-6.1f  工作%7.1f+%-6.1f  幽灵%7.1f+%-6.1f 面板H%6.1f 空白%4.1f %s\n",
			f.T.Milliseconds(), f.Dt.Milliseconds(), wOf(f, pzSide),
			f.R[pzStage].X, f.R[pzStage].W, f.R[pzChat].X, wOf(f, pzChat), f.R[pzWs].X, wOf(f, pzWs), f.R[pzGhost].X, wOf(f, pzGhost),
			f.R[pzPanel].H, f.Blank, f.Flags)
	}
	fmt.Print(sb.String())
}

// animCheck 通用断言:没有异常帧。
func animCheck(t *testing.T, name string, fr []AnimFrame) {
	t.Helper()
	if len(fr) < 4 {
		t.Fatalf("%s:动画帧太少(%d),探针没抓到动画", name, len(fr))
	}
	for _, f := range fr {
		if f.Flags != "" {
			t.Errorf("%s:%dms 异常帧:%s", name, f.T.Milliseconds(), f.Flags)
		}
	}
}

// animResize 模拟拖窗口边缘:宽度从 from 到 to,每帧(16 ms)变化 step 个像素。返回逐帧数据。
func animResize(t *testing.T, a *App, tt *ui.Tester, name string, from, to, step int) []AnimFrame {
	t.Helper()
	return animRun(t, a, tt, name, func() {
		tt.SetSize(from, 800)
		tt.Frame()
		a.AnimProbeKeep(true)
		w := from
		for w != to {
			if from < to {
				w = min(w+step, to)
			} else {
				w = max(w-step, to)
			}
			tt.SetSize(w, 800)
			tt.Advance(16 * time.Millisecond)
		}
	}, 100*time.Millisecond)
}

// 单项场景:无空白 / 无跳变 / 不掉帧(探针的逐帧判定),并校验起止宽度。
func TestAnimLayoutScenarios(t *testing.T) {
	a, tt := quickApp(t)
	a.SetMode(ModeAgent)
	tt.Frame()
	type sc struct {
		name string
		act  func()
		dur  time.Duration
	}
	run := func(s sc) []AnimFrame {
		fr := animRun(t, a, tt, s.name, s.act, s.dur)
		animCheck(t, s.name, fr)
		return fr
	}
	// 收起左栏:导航栏 → 0,对话栏同步变宽
	fr := run(sc{"Agent 收起左栏", func() { a.sideOpen = false }, 600 * time.Millisecond})
	if e := fr[len(fr)-1]; wOf(e, pzSide) > 1.5 {
		t.Errorf("收起左栏后导航栏应收到 0,实际 %.1f", wOf(e, pzSide))
	}
	run(sc{"Agent 展开左栏", func() { a.sideOpen = true }, 600 * time.Millisecond})
	run(sc{"Agent 展开右栏(工作区)", func() { a.toggleRight() }, 700 * time.Millisecond})
	run(sc{"Agent 收起右栏", func() { a.toggleRight() }, 700 * time.Millisecond})
	run(sc{"Agent→IDE", func() { a.SetMode(ModeIDE) }, 700 * time.Millisecond})
	fr = run(sc{"IDE 收起右栏(对话栏)", func() { a.toggleRight() }, 700 * time.Millisecond})
	if e := fr[len(fr)-1]; wOf(e, pzChat) > 2 {
		t.Errorf("IDE 收起对话栏后应收到 0,实际 %.1f", wOf(e, pzChat))
	}
	run(sc{"IDE 展开右栏", func() { a.toggleRight() }, 700 * time.Millisecond})
	run(sc{"IDE 收起底部面板", func() { a.ide.bottomOpen = false }, 500 * time.Millisecond})
	run(sc{"IDE 展开底部面板", func() { a.ide.bottomOpen = true }, 500 * time.Millisecond})
	run(sc{"IDE 收起左栏", func() { a.sideOpen = false }, 600 * time.Millisecond})
	run(sc{"IDE 展开左栏", func() { a.sideOpen = true }, 600 * time.Millisecond})
	run(sc{"IDE→Agent", func() { a.SetMode(ModeAgent) }, 700 * time.Millisecond})
}

// 拖窗口边缘(逐帧改窗口宽):跨过 1040 / 900 / 720 断点时导航栏、工作区、对话栏都是连续变化,
// 舞台右缘始终贴着窗口右缘,没有空白、没有一帧跳几百像素。
func TestAnimResize(t *testing.T) {
	a, tt := quickApp(t)
	a.SetMode(ModeAgent)
	tt.Frame()
	animCheck(t, "Agent 缩窗", animResize(t, a, tt, "Agent 缩窗 1280→700", 1280, 700, 12))
	animCheck(t, "Agent 放窗", animResize(t, a, tt, "Agent 放窗 700→1280", 700, 1280, 12))
	a.SetMode(ModeIDE)
	tt.Advance(time.Second)
	animCheck(t, "IDE 缩窗", animResize(t, a, tt, "IDE 缩窗 1280→700", 1280, 700, 12))
	animCheck(t, "IDE 放窗", animResize(t, a, tt, "IDE 放窗 700→1280", 700, 1280, 12))
	tt.SetSize(1280, 800)
}

// 导航栏逐行落下的延迟:严格递增(越靠下越晚),长列表后半段仍逐行往下排,总延迟封顶。
func TestNavDropDelay(t *testing.T) {
	prev := time.Duration(-1)
	for i := 0; i < 200; i++ {
		d := navDropDelay(i)
		if d < prev {
			t.Fatalf("第 %d 行延迟 %v 小于上一行 %v", i, d, prev)
		}
		if d > navDropCap {
			t.Fatalf("第 %d 行延迟 %v 超过封顶 %v", i, d, navDropCap)
		}
		prev = d
	}
	if navDropDelay(13) <= navDropDelay(12) || navDropDelay(30) <= navDropDelay(13) {
		t.Fatal("第 12 行之后不该同一时刻出场(储藏 / 图谱会整块蹦出)")
	}
	if navDropDelay(200) != navDropCap {
		t.Fatalf("很长的列表应封顶在 %v,实际 %v", navDropCap, navDropDelay(200))
	}
}
