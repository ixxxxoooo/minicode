package app

import (
	"github.com/ixxxxoooo/minicode/internal/diag"
)

var respLog = diag.L("resp")

// 窗口断点(设计稿 resp / arch 页,固定 1100 / 900 / 720):
//
//	≥1040  默认并排(右侧栏随窗口连续缩小,最小 360 / 300,主区始终 ≥ 440)
//	900–1039  Agent 工作区改浮层;IDE 导航栏改抽屉(⌘B 展开为浮层)
//	720–899  只剩主区:Agent 导航栏也改抽屉;IDE 对话栏收成图标条 + 浮层;标题栏 ⌘K 收成图标
//	<720   导航栏改抽屉;形态分段收成单图标
const (
	bpWide     = 1040 // = 导航栏 228 + 主区 440 + 右侧栏最小 360 + 间隙
	bpMid      = 900
	bpNarrow   = 720
	fitHyst    = 24
	palHidePrv = 820
	palNarrow  = 520
)

// autoFit 记录自动收起的状态(用户手动打开后不再算自动)。
type autoFit struct {
	w      float32
	side   bool
	right  bool
	wsOvl  bool // Agent:工作区从并排改为浮层
	dockRl bool // IDE:对话栏收成图标条
	mode   Mode
}

// winTier 返回当前窗口档位 0–3。
func (a *App) winTier() int {
	w := a.winW
	switch {
	case w >= bpWide:
		return 0
	case w >= bpMid:
		return 1
	case w >= bpNarrow:
		return 2
	default:
		return 3
	}
}

func (a *App) useWsOverlay() bool {
	return a.mode == ModeAgent && a.winTier() >= 1
}

func (a *App) useDockRail() bool {
	return a.mode == ModeIDE && !a.chatMax && a.winTier() >= 2
}

// useSideDrawer 导航栏改抽屉:Agent <900(只剩对话栏);IDE <1040(只剩工作区 + 对话栏,⌘B 展开为浮层)。
func (a *App) useSideDrawer() bool {
	if a.mode == ModeIDE {
		return a.winTier() >= 1
	}
	return a.winTier() >= 2
}

func (a *App) compactTitle() bool {
	return a.winTier() >= 2
}

func (a *App) compactModeSeg() bool {
	return a.winTier() >= 3
}

// pageNarrowMid 整页(设置 / 自定义 / 用量)在中等宽度下的紧凑布局。
func (a *App) pageNarrowMid() bool {
	return a.winW > 0 && a.winW < bpMid
}

// fitPanels 按断点自动收起 / 恢复次要区域;变宽只恢复曾被自动收起的项。
func (a *App) fitPanels() {
	w := a.winW
	if w <= 0 {
		return
	}
	f := &a.fit
	first, shrinking := f.w == 0, w < f.w
	f.w = w
	if f.mode != a.mode {
		*f = autoFit{w: w, mode: a.mode}
	}
	a.sideDrawer = a.useSideDrawer()

	// 用户在自动收起后又手动打开 → 清除自动标记
	if f.side && a.sideOpen {
		f.side = false
	}
	if f.right && a.rightOn && !a.useDockRail() && !a.useWsOverlay() {
		f.right = false
	}
	if f.wsOvl && !a.useWsOverlay() {
		f.wsOvl = false
	}
	if f.dockRl && !a.useDockRail() {
		f.dockRl = false
	}

	tier := a.winTier()
	switch {
	case a.useSideDrawer() && a.sideOpen && (first || shrinking):
		// 抽屉模式默认应收起(由标题栏按钮滑出);否则会像并排导航一样盖住主内容(cmd/shot w=700 即如此)。
		a.sideOpen, f.side = false, true
		respLog.Info("窗口变窄:导航抽屉收起", "w", w, "tier", tier)
	case tier >= 1 && a.mode == ModeIDE && a.sideOpen && (first || shrinking):
		a.sideOpen, f.side = false, true
		respLog.Info("窗口变窄:收起导航栏", "w", w, "tier", tier)
	case tier < 1 && f.side:
		a.sideOpen, f.side = true, false
		respLog.Info("窗口变宽:恢复导航栏", "w", w)
	case !a.useSideDrawer() && f.side && w > f.w:
		// 离开 <720 抽屉档:Agent 或宽窗恢复并排导航;IDE 中档仍保持收起。
		if a.winTier() < 1 || a.mode == ModeAgent {
			a.sideOpen, f.side = true, false
			respLog.Info("窗口变宽:恢复导航栏", "w", w, "tier", tier)
		} else {
			f.side = false
		}
	}

	switch {
	case tier >= 1 && a.mode == ModeAgent && a.winTier() == 1 && a.rightOn && !a.useWsOverlay() && (first || shrinking):
		// 仍算「开着工作区」,但并排改浮层
		f.wsOvl = true
		respLog.Info("窗口变窄:Agent 工作区改浮层", "w", w)
	case tier == 0 && f.wsOvl:
		f.wsOvl = false
	}

	switch {
	case a.useDockRail() && a.rightOn && !f.dockRl && (first || shrinking):
		f.dockRl, f.right = true, true
		respLog.Info("窗口变窄:IDE 对话栏收成图标条", "w", w)
	case !a.useDockRail() && f.dockRl:
		f.dockRl, f.right = false, false
	}

	// <900 Agent 若仍并排占不下:收起右侧(浮层模式下靠 wsFloat)
	needRightInline := float32(0)
	if a.mode == ModeAgent && !a.useWsOverlay() {
		needRightInline = 400 + 360
	}
	switch {
	case w < needRightInline && needRightInline > 0 && a.rightOn && (first || shrinking):
		a.rightOn, f.right = false, true
		respLog.Info("窗口变窄:收起 Agent 工作区", "w", w)
	case needRightInline > 0 && w >= needRightInline+fitHyst && f.right && !f.wsOvl:
		a.rightOn, f.right = true, false
	}

	if a.useWsOverlay() && a.rightOn {
		a.wsFloat = a.wsFloat || f.wsOvl
	}
	if a.useDockRail() && f.dockRl {
		a.dockFloat = a.dockFloat && a.rightOn
	}
	if !a.rightOn {
		a.wsFloat, a.dockFloat = false, false
	}
}
