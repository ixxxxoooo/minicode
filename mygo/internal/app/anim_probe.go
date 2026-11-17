package app

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
)

// 布局动画探针:每帧记下导航栏 / 舞台 / 对话栏 / 工作区 / 旧工作区幽灵 / 底部面板的矩形,
// 在一次「布局动画」(开合 / 形态切换 / 窗口缩放)里检查三件事,并落日志:
//
//	空白   舞台里有一段没被任何面板盖住(切换时闪出一条空白);
//	跳变   某个宽度 / 位置的逐帧变化量突然改变(二阶差分超阈值 = 从宽跳到窄或反过来);
//	掉帧   相邻两帧间隔超过 34 ms(低于 30 fps)。
//
// 日志(模块 anim):每次布局动画结束写一条 Info 汇总(帧数 / 时长 / 帧率 / 最大帧间隔 / 各区起止宽度 / 最大单帧位移 / 异常数),
// 异常帧写 Warn;设置环境变量 MINI_ANIMLOG=1 时每一帧都写一条 Info(字段含各区 x / 宽),
// 用来事后对着曲线看「宽度是否平滑」。无窗口测试 / cmd/shot 直接读 AnimFrames / AnimSummaries。
//
// 探针只读上一帧量到的矩形(Element.Bounds),不影响布局;没有动画时每帧只做几次矩形比较。

// 探针区域,下标即 AnimFrame.R 的下标。
const (
	pzStage = iota // 舞台(导航栏右边的整块)
	pzSide         // 导航栏
	pzChat         // 对话栏
	pzWs           // 工作区
	pzGhost        // 形态切换时被挤出舞台的旧工作区
	pzPanel        // 底部面板
	pzN
)

var probeNames = [pzN]string{"舞台", "导航栏", "对话栏", "工作区", "幽灵", "底部面板"}

// AnimFrame 是一帧的布局快照。
type AnimFrame struct {
	T     time.Duration // 距本次动画第一帧
	Dt    time.Duration // 距上一帧
	Win   float32       // 窗口宽
	R     [pzN]ui.Rect  // 各区矩形(上一帧量到的)
	Has   [pzN]bool     // 这一帧该区存在
	Inst  bool          // 拖分隔线 / 连续缩放窗口:直接跟随,不检查跳变
	Blank float32       // 舞台里没被盖住的横向长度
	Flags string        // 异常说明(空 = 正常)
}

// AnimSummary 是一次布局动画的汇总。
type AnimSummary struct {
	Frames     int
	Dur        time.Duration
	FPS        float64
	AvgDt, P95 time.Duration
	MaxDt      time.Duration
	Jumps      int
	BlankN     int
	Slow       int // 帧间隔 > 34 ms 的帧数
	Start, End [pzN]ui.Rect
	MaxStep    [pzN]float32 // 各区宽度(底部面板为高度)的最大单帧变化量
	Worst      string       // 最严重的一条异常
}

type animProbe struct {
	on      bool // 逐帧写 Info(MINI_ANIMLOG=1)
	cur     [pzN]ui.Rect
	seen    [pzN]bool
	prev    *AnimFrame
	prevD   [pzN]float32 // 上一帧各区宽度的变化量
	prevX   [pzN]float32
	frames  []AnimFrame
	start   time.Time
	last    time.Time // 上一帧时间
	moved   time.Time // 最近一次有变化
	busy    bool      // 正在一次布局动画里
	sums    []AnimSummary
	keep    bool // 保留逐帧数据(测试 / shot 读取);真实运行只留汇总,省内存
	suspend int  // >0 时不记录(测试里关掉)
}

var probeOnEnv = os.Getenv("MINI_ANIMLOG") == "1"

// probe 记下某个区域本帧的位置(读上一帧量到的矩形)。el 无效 / 还没布局过时当作「不存在」。
func (a *App) probe(z int, el ui.Element) {
	r := el.Bounds()
	if r.W <= 0 && r.H <= 0 {
		return
	}
	a.pr.cur[z], a.pr.seen[z] = r, true
}

// AnimProbeKeep 让探针保留逐帧数据(测试 / cmd/shot 用),并清空已有记录。
func (a *App) AnimProbeKeep(on bool) {
	a.pr.keep = on
	a.pr.frames, a.pr.sums = nil, nil
}

// AnimFrames 返回最近一次(进行中则当前)布局动画的逐帧数据。
func (a *App) AnimFrames() []AnimFrame { return a.pr.frames }

// AnimSummaries 返回已结束的布局动画汇总(时间顺序)。
func (a *App) AnimSummaries() []AnimSummary { return a.pr.sums }

// probeEnd 每帧在界面构建完成后调用:比较与上一帧的差异,必要时开始 / 延续 / 结束一次动画记录。
func (a *App) probeEnd(now time.Time) {
	p := &a.pr
	if p.suspend > 0 {
		return
	}
	cur, seen := p.cur, p.seen
	p.cur, p.seen = [pzN]ui.Rect{}, [pzN]bool{}
	if a.screen != ScreenWork && a.screen != ScreenWelcome {
		p.prev = nil
		return
	}
	// 与上一帧比较:有任何区域的位置 / 大小变了才算「在动」
	moved := false
	if p.prev != nil {
		for i := 0; i < pzN; i++ {
			if seen[i] != p.prev.Has[i] || (seen[i] && rectDiff(cur[i], p.prev.R[i]) > .05) {
				moved = true
				break
			}
		}
		if p.prev.Win != a.winW {
			moved = true
		}
	}
	idle := p.moved.IsZero() || now.Sub(p.moved) > 150*time.Millisecond
	if moved {
		if !p.busy && idle {
			p.busy, p.start, p.frames, p.last = true, now, p.frames[:0], time.Time{}
			p.prevD = [pzN]float32{}
			p.prevX = [pzN]float32{}
		}
		p.moved = now
	}
	if !p.busy {
		f := AnimFrame{Win: a.winW, R: cur, Has: seen}
		p.prev = &f
		return
	}
	if !p.last.IsZero() && !now.After(p.last) { // 同一时刻的重复构建(一次刷新里连续构建几遍)只算一帧
		return
	}
	f := AnimFrame{T: now.Sub(p.start), Win: a.winW, R: cur, Has: seen, Inst: a.instantLayout()}
	if !p.last.IsZero() {
		f.Dt = now.Sub(p.last)
	}
	p.last = now
	a.probeCheck(&f)
	p.prev = &f
	p.frames = append(p.frames, f)
	if probeOnEnv {
		a.probeLogFrame(&f)
	}
	if f.Flags != "" {
		diag.L("anim").Warn("布局动画异常帧", "t", f.T.Round(time.Millisecond), "dt", f.Dt.Round(time.Microsecond), "异常", f.Flags,
			"窗口", f.Win, "导航栏", wOf(f, pzSide), "对话栏", spanOf(f, pzChat), "工作区", spanOf(f, pzWs))
	}
	if now.Sub(p.moved) > 150*time.Millisecond || len(p.frames) >= 4000 {
		a.probeFinish()
		return
	}
	if a.win != nil || a.driveFrames {
		a.frameSoon() // 等到「静止满 150 ms」才能结算汇总,中途不能没有帧
	}
}

func rectDiff(a, b ui.Rect) float32 {
	return max(absF(a.X-b.X), absF(a.Y-b.Y), absF(a.W-b.W), absF(a.H-b.H))
}

func wOf(f AnimFrame, z int) float32 {
	if !f.Has[z] {
		return 0
	}
	return f.R[z].W
}

func spanOf(f AnimFrame, z int) string {
	if !f.Has[z] {
		return "-"
	}
	return fmt.Sprintf("%.1f+%.1f", f.R[z].X, f.R[z].W)
}

// probeCheck 对一帧做空白 / 跳变 / 掉帧检查,结果写进 f.Flags。
func (a *App) probeCheck(f *AnimFrame) {
	p := &a.pr
	var fl []string
	gap := a.tk.Scheme.GapPane
	// 1) 空白:舞台里被 对话栏 / 工作区 / 幽灵 盖住的并集,到舞台两端的缺口
	if f.Has[pzStage] {
		st := f.R[pzStage]
		type iv struct{ l, r float32 }
		var ivs []iv
		for _, z := range []int{pzChat, pzWs, pzGhost} {
			if f.Has[z] && f.R[z].W > .5 {
				ivs = append(ivs, iv{f.R[z].X, f.R[z].X + f.R[z].W})
			}
		}
		sort.Slice(ivs, func(i, j int) bool { return ivs[i].l < ivs[j].l })
		edge := st.X
		blank := float32(0)
		for _, v := range ivs {
			if v.l > edge+gap+.5 { // 面板之间允许一个面板间距
				blank += v.l - edge - gap
			}
			edge = max(edge, v.r)
		}
		if edge < st.X+st.W-.5 {
			blank += st.X + st.W - edge
		}
		if len(ivs) == 0 {
			blank = st.W
		}
		f.Blank = blank
		if blank > 1.5 {
			fl = append(fl, fmt.Sprintf("空白%.1fpx", blank))
		}
		// 舞台右缘应贴着窗口右缘(减去外边距)
		// (取的是上一帧的矩形,所以对照上一帧的窗口宽)
		ref := a.winW
		if p.prev != nil {
			ref = p.prev.Win
		}
		if ref > 0 && absF(ref-gap-(st.X+st.W)) > 1.5 && f.Has[pzSide] {
			fl = append(fl, fmt.Sprintf("舞台右缘偏离%.1fpx", ref-gap-(st.X+st.W)))
		}
	}
	// 1b) 底部面板的右缘不能落在工作区右缘之内(面板在工作区里,宽度同帧一致);
	// 偏窄 = 终端比编辑器窄一截、旁边露出空白(曾因给面板宽度挂了元素过渡而出现)。
	// 偏宽(d<0)不算:工作区在收窄动画里,面板内容保持原宽被工作区裁切,和导航栏收起时一样,不是空白。
	if f.Has[pzPanel] && f.Has[pzWs] && f.R[pzPanel].H > 1.5 && f.R[pzWs].W > 40 && !f.Has[pzGhost] {
		if d := (f.R[pzWs].X + f.R[pzWs].W) - (f.R[pzPanel].X + f.R[pzPanel].W); d > 1.5 {
			fl = append(fl, fmt.Sprintf("底部面板右缘偏离工作区%.1fpx", d))
		}
	}
	// 2) 跳变:宽度 / X / 高度的二阶差分
	// (拖分隔条是逐帧跟手的「瞬时布局」,不检查;拖窗口边缘则检查,容差随窗口每帧位移放宽)
	winMove := float32(0)
	if p.prev != nil {
		winMove = absF(f.Win - p.prev.Win)
	}
	if p.prev != nil && (winMove > 0 || (!f.Inst && !p.prev.Inst)) && f.Dt > 0 {
		norm := float32(f.Dt) / float32(16*time.Millisecond)
		// 帧间隔短于 16 ms(窗口缩放时一次刷新会连构建几遍)不收紧阈值;形态曲线(ease-morph)前段本来就陡,
		// 窗口同时在变时再放宽一档
		norm = min(max(norm, 1), 3) + winMove*3/40
		if winMove > 0 {
			norm *= 2
		}
		for _, z := range []int{pzSide, pzChat, pzWs, pzPanel} {
			if !f.Has[z] || !p.prev.Has[z] {
				continue
			}
			var d float32
			if z == pzPanel {
				d = f.R[z].H - p.prev.R[z].H
			} else {
				d = f.R[z].W - p.prev.R[z].W
			}
			if absF(d-p.prevD[z]) > 40*norm && p.prev != nil && len(p.frames) >= 2 {
				fl = append(fl, fmt.Sprintf("%s跳变%.1fpx", probeNames[z], d-p.prevD[z]))
			}
			p.prevD[z] = d
		}
	}
	// 3) 掉帧
	if f.Dt > 34*time.Millisecond {
		fl = append(fl, fmt.Sprintf("掉帧%dms", f.Dt.Milliseconds()))
	}
	f.Flags = strings.Join(fl, ",")
}

func (a *App) probeLogFrame(f *AnimFrame) {
	diag.L("anim").Info("帧", "t", f.T.Round(time.Millisecond), "dt", f.Dt.Round(100*time.Microsecond), "窗口", f.Win,
		"导航栏", spanOf(*f, pzSide), "舞台", spanOf(*f, pzStage), "对话栏", spanOf(*f, pzChat), "工作区", spanOf(*f, pzWs),
		"幽灵", spanOf(*f, pzGhost), "底部面板高", func() float32 {
			if f.Has[pzPanel] {
				return f.R[pzPanel].H
			}
			return 0
		}(), "空白", f.Blank, "异常", f.Flags)
}

// probeFinish 结算一次布局动画:写汇总日志、存汇总。
func (a *App) probeFinish() {
	p := &a.pr
	p.busy = false
	n := len(p.frames)
	if n < 3 { // 只动了一两帧(比如窗口宽度抖了一下)不值得汇总
		return
	}
	s := AnimSummary{Frames: n, Dur: p.frames[n-1].T}
	dts := make([]time.Duration, 0, n)
	for i, f := range p.frames {
		if i > 0 {
			dts = append(dts, f.Dt)
			s.AvgDt += f.Dt
			s.MaxDt = max(s.MaxDt, f.Dt)
			if f.Dt > 34*time.Millisecond {
				s.Slow++
			}
			for z := 0; z < pzN; z++ {
				if f.Has[z] && p.frames[i-1].Has[z] {
					d := f.R[z].W - p.frames[i-1].R[z].W
					if z == pzPanel {
						d = f.R[z].H - p.frames[i-1].R[z].H
					}
					s.MaxStep[z] = max(s.MaxStep[z], absF(d))
				}
			}
		}
		if f.Blank > 1.5 {
			s.BlankN++
			if s.Worst == "" {
				s.Worst = f.Flags
			}
		}
		if strings.Contains(f.Flags, "跳变") {
			s.Jumps++
			if s.Worst == "" || !strings.Contains(s.Worst, "跳变") {
				s.Worst = f.Flags
			}
		}
	}
	if len(dts) > 0 {
		s.AvgDt /= time.Duration(len(dts))
		sort.Slice(dts, func(i, j int) bool { return dts[i] < dts[j] })
		s.P95 = dts[min(len(dts)-1, len(dts)*95/100)]
		if s.AvgDt > 0 {
			s.FPS = float64(time.Second) / float64(s.AvgDt)
		}
	}
	s.Start, s.End = p.frames[0].R, p.frames[n-1].R
	lg := diag.L("anim")
	kv := []any{"帧数", s.Frames, "时长", s.Dur.Round(time.Millisecond), "帧率", fmt.Sprintf("%.0f", s.FPS),
		"平均帧间隔", s.AvgDt.Round(100 * time.Microsecond), "p95", s.P95.Round(100 * time.Microsecond), "最大帧间隔", s.MaxDt.Round(100 * time.Microsecond),
		"掉帧", s.Slow, "跳变", s.Jumps, "空白帧", s.BlankN,
		"导航栏", fmt.Sprintf("%.0f→%.0f", s.Start[pzSide].W, s.End[pzSide].W),
		"对话栏", fmt.Sprintf("%.0f→%.0f", s.Start[pzChat].W, s.End[pzChat].W),
		"工作区", fmt.Sprintf("%.0f→%.0f", s.Start[pzWs].W, s.End[pzWs].W),
		"底部面板", fmt.Sprintf("%.0f→%.0f", s.Start[pzPanel].H, s.End[pzPanel].H),
		"最大单帧位移", fmt.Sprintf("导航栏%.1f 对话栏%.1f 工作区%.1f 面板%.1f", s.MaxStep[pzSide], s.MaxStep[pzChat], s.MaxStep[pzWs], s.MaxStep[pzPanel])}
	if s.Jumps+s.BlankN+s.Slow > 0 {
		lg.Warn("布局动画结束(有异常)", append(kv, "最严重", s.Worst)...)
	} else {
		lg.Info("布局动画结束", kv...)
	}
	p.sums = append(p.sums, s)
	if len(p.sums) > 64 {
		p.sums = p.sums[len(p.sums)-64:]
	}
	if !p.keep {
		p.frames = p.frames[:0]
	}
}
