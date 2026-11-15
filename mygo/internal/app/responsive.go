package app

// 窗口宽度自适应(设计稿「响应式」页的窗口宽度断点):窗口变窄放不下时,按优先级自动收起次要区域,
// 变宽后只恢复「自动收起的」那些;用户手动开合优先,不会被覆盖。
//
//	Agent 形态  对话栏是主角:先收导航栏(宽 < 导航栏 + 对话栏 400 + 工作区 360),再收右侧工作区(宽 < 400 + 360)
//	IDE 形态    工作区是主角:先收导航栏(宽 < 导航栏 + 工作区 480 + 对话栏),再收右侧对话栏(宽 < 480 + 对话栏)
//
// 恢复带 24px 的回差,避免在阈值附近抖动。只在宽度「继续变窄」或第一帧才会触发收起。
const (
	fitChatMin = 400 // Agent 对话栏最小宽度
	fitWsMin   = 360 // Agent 工作区最小宽度
	fitMainMin = 480 // IDE 工作区最小宽度
	fitHyst    = 24
)

// autoFit 记录自动收起的状态。
type autoFit struct {
	w     float32 // 上一帧窗口宽度
	side  bool    // 导航栏是被自动收起的
	right bool    // 右侧栏是被自动收起的
	mode  Mode    // right 标记属于哪个形态(形态切换后作废)
}

// fitNeeds 返回当前形态下「导航栏 + 右侧栏都开」与「只开右侧栏」所需的最小窗口宽度。
func fitNeeds(mode Mode, sideW, dockW float32) (withSide, withRight float32) {
	if mode == ModeIDE {
		return sideW + fitMainMin + dockW, fitMainMin + dockW
	}
	return sideW + fitChatMin + fitWsMin, fitChatMin + fitWsMin
}

// fitPanels 每帧在构建界面之前调用:按窗口宽度收起 / 恢复导航栏与右侧栏。
func (a *App) fitPanels() {
	w := a.winW
	if w <= 0 {
		return
	}
	f := &a.fit
	first, shrinking := f.w == 0, w < f.w
	f.w = w
	if f.mode != a.mode {
		f.right, f.mode = false, a.mode
	}
	needSide, needRight := fitNeeds(a.mode, a.sideW(), a.dockW())

	// 用户在自动收起之后又自己打开了:不再算「自动收起」
	if f.side && a.sideOpen {
		f.side = false
	}
	if f.right && a.rightOn {
		f.right = false
	}
	switch {
	case w < needSide && a.sideOpen && (first || shrinking):
		a.sideOpen, f.side = false, true
		log.Info("窗口变窄:自动收起导航栏", "w", w)
	case w >= needSide+fitHyst && f.side:
		a.sideOpen, f.side = true, false
		log.Info("窗口变宽:恢复导航栏", "w", w)
	}
	switch {
	case w < needRight && a.rightOn && (first || shrinking):
		a.rightOn, f.right = false, true
		log.Info("窗口变窄:自动收起右侧栏", "w", w, "mode", a.mode)
	case w >= needRight+fitHyst && f.right:
		a.rightOn, f.right = true, false
		log.Info("窗口变宽:恢复右侧栏", "w", w, "mode", a.mode)
	}
}
