package app

import "time"

// syncOverlayMotion 在浮层开关变化时记下入场时刻(抽屉滑入动画)。
func (a *App) syncOverlayMotion() {
	now := time.Now()
	if a.useWsOverlay() && a.wsFloat {
		if a.wsOvlAt.IsZero() {
			a.wsOvlAt = now
		}
	} else {
		a.wsOvlAt = time.Time{}
	}
	if a.useSideDrawer() && a.sideOpen {
		if a.sideOvlAt.IsZero() {
			a.sideOvlAt = now
		}
	} else {
		a.sideOvlAt = time.Time{}
	}
	if a.useDockRail() && a.dockFloat && a.rightOn {
		if a.dockOvlAt.IsZero() {
			a.dockOvlAt = now
		}
	} else {
		a.dockOvlAt = time.Time{}
	}
}
