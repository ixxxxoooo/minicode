package app

import (
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/backend"
	"github.com/ixxxxoooo/minicode/internal/core"
)

// Claude 连接的预热与空闲释放。
//
// 背景(实测 + 业界做法,详见 AGENTS.md「连接预热」一节):
//   - 惰性启动把「拉起 claude 进程 + 握手」放在点发送之后,首响应要多等这一段;恢复会话、装了插件 / MCP 的项目更久。
//   - 官方 Agent SDK 的 startup() 就是为此而设:在拿到提示词之前先 spawn + 握手,首问快一个数量级。
//   - 反面教材:Zed 打开项目就给恢复的草稿线程起 agent(一个窗口 ~335 MB + MCP);VS Code 扩展每个面板一个
//     进程且从不回收(有人 3 个面板留下 29 个进程、7.4 GB)。所以预热必须「只给用户正看着的那个会话」且「闲置要回收」。
//
// 策略(三道闸):
//  1. 触发 = 意图:会话成为前台(打开 / 新建 / 启动恢复)且窗口在前台,去抖 350 ms 后预热——快速切换、方向键翻列表不会起进程;
//     输入框里开始打字、窗口从后台回到前台也会补预热(空闲释放之后用户回来)。
//  2. 数量:只有前台会话持有「空闲进程」;切走时空闲的立刻结束,正在跑回合的转后台保活到回合结束(App.parkOrClose)。
//  3. 回收:前台闲置超过设定时间(默认 10 分钟,窗口在后台时 ⅓)就结束进程,会话保留,下次发送自动 --resume;
//     回合进行中 / 有后台任务 / 有待处理请求时后端拒绝回收。
const (
	pWarm = "chat.warm" // 0 打开会话时(默认)· 1 输入时 · 2 不预热
	pIdle = "chat.idle" // 0 5 分钟 · 1 10 分钟(默认)· 2 30 分钟 · 3 不释放

	warmDebounce  = 350 * time.Millisecond
	warmRelRetry  = 60 * time.Second // 后端拒绝回收后多久再试
	warmBgDivisor = 3                // 窗口在后台时空闲时限除以这个数
)

type warmCtl struct {
	be      backend.Backend
	due     time.Time   // 排期的预热时刻(零 = 没有排期)
	sent    bool        // 本周期已发出过预热(释放 / 退出后清零)
	lastAct time.Time   // 最近一次「用户在用」:发送 / 回合开始与结束 / 打字 / 切回
	relTry  time.Time   // 最近一次发出回收命令
	fgWas   bool        // 上一次看到的窗口前台状态
	draftN  int         // 上一次的草稿长度(变化 = 在打字)
	fgHook  func() bool // 测试注入:覆盖「窗口是否在前台」
}

func (a *App) warmPolicy() int { return a.prefs.Int(pWarm, 0) }

// idleLimit 空闲多久后释放;0 = 不释放。
func (a *App) idleLimit() time.Duration {
	d := []time.Duration{5 * time.Minute, 10 * time.Minute, 30 * time.Minute, 0}
	i := a.prefs.Int(pIdle, 1)
	if i < 0 || i >= len(d) {
		i = 1
	}
	return d[i]
}

// warmEligible 只有真实的 claude 后端有进程可预热(演示 / 测试后端没有)。
func (a *App) warmEligible() bool { return a.be != nil && a.be.Name() == "claude" }

// warmAttach 新后端成为前台:重置状态,「打开会话时」策略下排期预热。
func (a *App) warmAttach(be backend.Backend) {
	now := a.now()
	a.warm = warmCtl{be: be, lastAct: now, fgWas: a.warmFg(), fgHook: a.warm.fgHook}
	if a.warmEligible() && a.warmPolicy() == 0 {
		// 打开会话 = 立刻连接(不再去抖):被切走的空闲进程会随即结束,同一时刻最多一个空闲进程。
		// 窗口不在前台就排期,回前台再起(别给没人看的窗口起进程)。
		a.warm.due = now
		if a.warmFg() {
			a.warmFire("打开会话")
		}
	}
}

// warmAdopt 接回一个已经在后台跑着的后端:进程是活的,不需要预热,只重置空闲计时。
func (a *App) warmAdopt(be backend.Backend) {
	a.warm = warmCtl{be: be, sent: true, lastAct: a.now(), fgWas: a.warmFg(), fgHook: a.warm.fgHook}
}

// warmFg 窗口是否在前台(测试可注入)。
func (a *App) warmFg() bool {
	if a.warm.fgHook != nil {
		return a.warm.fgHook()
	}
	return a.appInForeground()
}

func (a *App) armWarmTimer(d time.Duration) {
	if a.win == nil {
		return // 测试 / 截图没有窗口:由测试直接调 warmTick
	}
	time.AfterFunc(d+10*time.Millisecond, func() { a.win.Update(a.warmTick) })
}

// noteActive 记一次「用户在用」,推迟空闲释放。
func (a *App) noteActive() {
	if a.warm.be == a.be {
		a.warm.lastAct = a.now()
	}
}

// warmFire 立刻发预热(进程已在连接 / 已连接则不重复)。
func (a *App) warmFire(why string) {
	w := &a.warm
	if !a.warmEligible() || w.be != a.be || a.st == nil {
		return
	}
	w.due = time.Time{}
	if a.st.Conn == core.ConnStarting || a.st.Conn == core.ConnReady {
		w.sent = true
		return
	}
	w.sent = true
	log.Info("预热 claude 连接", "原因", why, "会话", a.curSess)
	a.be.Send(core.CmdWarmup{})
}

// warmTypingSignal 用户在输入框里打字:记活跃;还没连接就立刻补预热(「打开会话时」被释放过、或「输入时」策略)。
func (a *App) warmTypingSignal() {
	n := len(strings.TrimSpace(a.draft))
	if n != a.warm.draftN {
		a.warm.draftN = n
		if n > 0 {
			a.noteActive()
		}
	}
	if n == 0 || a.warmPolicy() == 2 || a.warm.sent || !a.warmEligible() {
		return
	}
	a.warmFire("输入中")
}

// WarmTick 供 main 的节拍调用。
func (a *App) WarmTick() {
	a.warmTick()
	a.reapParked()
}

// warmTick 每帧(很便宜)+ 定时器 + 15 秒节拍调用:到点预热、回前台补预热、空闲释放。
func (a *App) warmTick() {
	w := &a.warm
	if w.be == nil || w.be != a.be || a.st == nil || !a.warmEligible() {
		return
	}
	now := a.now()
	fg := a.warmFg()
	if fg && !w.fgWas { // 窗口回到前台
		w.lastAct = now
		if a.warmPolicy() == 0 && !w.sent && w.due.IsZero() {
			w.due = now.Add(warmDebounce)
			a.armWarmTimer(warmDebounce)
		}
	}
	w.fgWas = fg
	if !w.due.IsZero() && !now.Before(w.due) {
		if fg { // 窗口不在前台就一直等,回前台再起(别给没人看的窗口起进程)
			a.warmFire("打开会话")
		}
	}
	a.maybeRelease(now, fg)
}

func (a *App) maybeRelease(now time.Time, fg bool) {
	w := &a.warm
	limit := a.idleLimit()
	if limit == 0 || a.st.Conn != core.ConnReady || a.st.Phase != core.PhaseIdle || a.q.len() > 0 {
		return
	}
	if !fg {
		limit /= warmBgDivisor
	}
	if now.Sub(w.lastAct) < limit || (!w.relTry.IsZero() && now.Sub(w.relTry) < warmRelRetry) {
		return
	}
	if len(a.st.RunningTasks()) > 0 { // 后台任务随进程一起死,有在跑的就不释放
		return
	}
	w.relTry = now
	log.Info("空闲释放 claude 连接", "闲置", now.Sub(w.lastAct).Round(time.Second), "窗口前台", fg, "会话", a.curSess)
	a.be.Send(core.CmdRelease{Why: "闲置 " + now.Sub(w.lastAct).Round(time.Second).String()})
}

// warmOnEvent 事件归约之后:回合开始 / 结束算活跃;进程没了(释放 / 退出)就允许再次预热。
func (a *App) warmOnEvent(ev core.Event) {
	switch e := ev.(type) {
	case core.EvTurnStarted, core.EvTurnEnded:
		a.noteActive()
	case core.EvConn:
		if e.State == core.ConnNone || e.State == core.ConnFailed {
			a.warm.sent = false
		}
		if e.State == core.ConnReady {
			a.noteActive() // 握手完成算一次活跃:预热后立刻开始计空闲
		}
	}
}
