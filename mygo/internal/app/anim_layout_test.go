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

// 底部终端收起 / 展开:内容(标签条)要保留到高度走完,不能一收起就消失;
// 透明度只在最后一半高度里变化,收起途中面板高度过半时仍然完全不透明。
func TestAnimBottomPanelCollapseKeepsContent(t *testing.T) {
	a, tt := quickApp(t)
	a.SetMode(ModeIDE)
	a.ide.bottomOpen, a.ide.bottomTab = true, "term"
	t.Cleanup(a.CloseTerminals)
	kit.SetClock(tt.Now)
	t.Cleanup(func() { kit.SetClock(nil) })
	tt.Advance(1500 * time.Millisecond)
	openN := len(tt.Texts())
	a.ide.bottomOpen = false
	tt.Frame()
	seen := 0
	for el := time.Duration(0); el < 240*time.Millisecond; el += 16 * time.Millisecond {
		tt.Advance(16 * time.Millisecond)
		if len(tt.Texts()) >= openN { // 面板内容的文字还在(界面文字数没少)
			seen++
		}
	}
	// 高度动画 260 ms:前 240 ms 内标签条应一直在(曾经 120 ms 就淡没了,只剩一条空底色)
	if seen < 14 {
		t.Fatalf("收起动画 240 ms 里只有 %d/15 帧还看得到面板内容,面板像是突然消失", seen)
	}
	tt.Advance(600 * time.Millisecond)
	if len(tt.Texts()) >= openN {
		t.Fatal("收起完成后面板内容应已移除(界面文字应变少)")
	}
}

// 两个形态各自记住导航栏的开合:IDE(左栏开)切到 Agent(左栏收起)、再切回去,
// 形态切换的传送带和导航栏收起 / 展开同时进行,舞台里任何一帧都不能露出空白。
func TestAnimModeSwitchSideDiffers(t *testing.T) {
	a, tt := quickApp(t)
	a.SetMode(ModeAgent)
	tt.Frame()
	a.sideOpen = false // Agent 记住「左栏收起」
	tt.Advance(time.Second)
	a.SetMode(ModeIDE)
	tt.Advance(time.Second)
	a.sideOpen, a.ide.bottomOpen = true, true // IDE 左栏开、底部面板开(用户截图的状态)
	tt.Advance(time.Second)
	animCheck(t, "IDE→Agent(左栏开→收)", animRun(t, a, tt, "IDE→Agent(左栏开→收)", func() { a.SetMode(ModeAgent) }, 800*time.Millisecond))
	animCheck(t, "Agent→IDE(左栏收→开)", animRun(t, a, tt, "Agent→IDE(左栏收→开)", func() { a.SetMode(ModeIDE) }, 800*time.Millisecond))
}

// beltChain 断言平移中「对话栏 | 工作区」首尾相接:没有被挤出舞台的幽灵,工作区左缘紧贴对话栏右缘(隔一个面板间距),
// 工作区右缘贴着舞台右缘。回归:IDE(对话栏收起)→ Agent 时,工作区曾一开始就摆在终点位置,
// 旧工作区(幽灵)留在原地被对话栏盖住,中间露出一块多余的内容。
func beltChain(t *testing.T, a *App, name string, fr []AnimFrame) {
	t.Helper()
	gap := a.tk.Scheme.GapPane
	for _, f := range fr {
		if f.Has[pzGhost] {
			t.Errorf("%s:%dms 不该有幽灵(同一个工作区在被对话栏挤过去)", name, f.T.Milliseconds())
		}
		if !f.Has[pzChat] || !f.Has[pzWs] || f.R[pzChat].W < 1 || f.R[pzWs].W < 1 {
			continue
		}
		if f.R[pzChat].X >= f.R[pzWs].X { // 动画结束后 IDE 的收起对话栏是停在右缘的 1 px 占位,不在检查范围
			continue
		}
		right := f.R[pzChat].X + f.R[pzChat].W
		if d := f.R[pzWs].X - right; d < -1.5 || d > gap+1.5 {
			t.Errorf("%s:%dms 工作区左缘 %.1f 没有贴着对话栏右缘 %.1f(间距 %.0f)", name, f.T.Milliseconds(), f.R[pzWs].X, right, gap)
		}
	}
}

// IDE(对话栏收起、工作区占满)→ Agent:对话框从左边长出来,把工作区向右挤、同时压窄;反向亦然。
func TestAnimBeltPushWorkspace(t *testing.T) {
	a, tt := quickApp(t)
	a.SetMode(ModeAgent)
	tt.Frame()
	a.rightOn = true // Agent 记住「工作区开」
	tt.Advance(time.Second)
	a.SetMode(ModeIDE)
	tt.Advance(time.Second)
	a.rightOn, a.sideOpen = false, true // IDE:对话栏收起、左栏开(用户截图的状态)
	tt.Advance(time.Second)
	fr := animRun(t, a, tt, "IDE(无对话栏)→Agent", func() { a.SetMode(ModeAgent) }, 800*time.Millisecond)
	animCheck(t, "IDE(无对话栏)→Agent", fr)
	beltChain(t, a, "IDE(无对话栏)→Agent", fr)
	fr = animRun(t, a, tt, "Agent→IDE(无对话栏)", func() { a.SetMode(ModeIDE) }, 800*time.Millisecond)
	animCheck(t, "Agent→IDE(无对话栏)", fr)
	beltChain(t, a, "Agent→IDE(无对话栏)", fr)
}

// Agent 只有会话(工作区收起)↔ IDE 只有编辑器(对话栏收起):同样是对话栏把编辑器挤过去并压窄,
// 编辑器不能一开始就摆在终点位置被对话栏盖着(回归:要求两个形态的工作区都开着才走「挤过去」,这种组合走了刚性公式)。
func TestAnimBeltPushWorkspaceAgentNoWs(t *testing.T) {
	a, tt := quickApp(t)
	a.SetMode(ModeAgent)
	tt.Frame()
	a.rightOn = false // Agent:只有会话
	tt.Advance(time.Second)
	a.SetMode(ModeIDE)
	tt.Advance(time.Second)
	a.rightOn, a.sideOpen = false, true // IDE:只有编辑器
	tt.Advance(time.Second)
	fr := animRun(t, a, tt, "IDE(只有编辑器)→Agent(只有会话)", func() { a.SetMode(ModeAgent) }, 800*time.Millisecond)
	animCheck(t, "IDE→Agent(无工作区)", fr)
	beltChain(t, a, "IDE→Agent(无工作区)", fr)
	fr = animRun(t, a, tt, "Agent(只有会话)→IDE(只有编辑器)", func() { a.SetMode(ModeIDE) }, 800*time.Millisecond)
	animCheck(t, "Agent→IDE(无工作区)", fr)
	beltChain(t, a, "Agent→IDE(无工作区)", fr)
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
