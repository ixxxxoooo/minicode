package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// overlayLayer 在根上绘制窄窗浮层:Agent 工作区、IDE 对话栏、导航抽屉。
func (a *App) overlayLayer(c *ui.Context) {
	a.syncOverlayMotion()
	if a.useSideDrawer() && a.sideOpen {
		a.sideDrawerOverlay(c)
	}
	if a.useWsOverlay() && a.rightOn && a.wsFloat {
		a.wsOverlay(c)
	}
	if a.useDockRail() && a.dockFloat && a.rightOn {
		a.dockOverlay(c)
	} else if a.useDockRail() && a.rightOn && !a.dockFloat {
		a.dockRail(c)
	}
}

func (a *App) wsOverlay(c *ui.Context) {
	t := a.tk
	w := min(a.wsRW(), a.winW*0.9)
	if w < 360 {
		w = min(360, a.winW-24)
	}
	ui.Box(c).Absolute().Left(0).Right(0).Top(0).Bottom(0).
		Background(t.Scrim).
		OnClick(func() { a.wsFloat = false }).
		Children(func() {
			col := ui.Column(c).Absolute().Right(0).Top(0).Bottom(0).Width(w).
				Background(t.BG).BorderWidth(1, 0, 0, 0).BorderColor(t.Bd1)
			for _, s := range t.Shadow3() {
				col.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
			}
			kit.DrawerSlide(col, a.wsOvlAt, true, a.frameSoon)
			col.Children(func() { a.workspace(c) })
		})
}

func (a *App) dockRail(c *ui.Context) {
	t := a.tk
	ui.Column(c).Absolute().Right(8).Top(theme.HBar + 8).Width(40).Gap(6).Children(func() {
		kit.Ib(c, "bot", kit.IbOpt{Tip: "对话栏", On: a.dockFloat}).OnClick(func() {
			a.dockFloat = true
		})
		if a.st != nil && len(a.st.Items) > 0 {
			kit.Ib(c, "chat", kit.IbOpt{Small: true, Tip: "会话"}).OnClick(func() { a.dockFloat = true })
		}
		_ = t
	})
}

func (a *App) dockOverlay(c *ui.Context) {
	t := a.tk
	w := min(a.dockW(), a.winW*0.92)
	if w < 300 {
		w = min(300, a.winW-24)
	}
	ui.Box(c).Absolute().Left(0).Right(0).Top(0).Bottom(0).
		Background(t.Scrim).
		OnClick(func() { a.dockFloat = false }).
		Children(func() {
			col := ui.Column(c).Absolute().Right(0).Top(0).Bottom(0).Width(w).
				Background(t.BG).BorderWidth(1, 0, 0, 0).BorderColor(t.Bd1)
			kit.DrawerSlide(col, a.dockOvlAt, true, a.frameSoon)
			col.Children(func() { a.chatPane(c, 0) })
		})
}

func (a *App) sideDrawerOverlay(c *ui.Context) {
	t := a.tk
	w := a.sideW()
	ui.Box(c).Absolute().Left(0).Right(0).Top(0).Bottom(0).
		Background(t.Scrim).
		OnClick(func() { a.sideOpen = false }).
		Children(func() {
			drawer := ui.Column(c).Absolute().Left(0).Top(0).Bottom(0).Width(w).
				Background(t.Chrome).BorderWidth(0, 1, 0, 0).BorderColor(t.Bd2)
			kit.DrawerSlide(drawer, a.sideOvlAt, false, a.frameSoon)
			drawer.Children(func() {
				innerW := w
				ui.Column(c).Width(innerW).Grow(1).MinHeight(0).Children(func() {
					a.sideTop(c)
					ui.Column(c).Grow(1).MinHeight(0).PaddingX(6).Clip().Children(func() {
						switch a.NavID() {
						case "chat":
							a.sessions(c)
						case "files":
							a.filesView(c)
						case "search":
							a.searchView(c)
						case "scm":
							a.scmView(c)
						}
					})
					if a.tk.Scheme.ID != "c" {
						a.sideFoot(c)
					}
				})
			})
		})
}

// revealWorkspace 窄窗 Agent 开标签时展开浮层。
func (a *App) openWsOverlay() {
	if a.useWsOverlay() {
		a.wsFloat = true
	}
}
