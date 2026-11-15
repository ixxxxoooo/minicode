package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 偏好键(设计稿 brand-spec / settings.js::brandGroups);Dock / 托盘桥由其它模块接。
const (
	pAppStyle  = "appearance.appIcon"
	pAppMode   = "appearance.appIconMode"
	pAppTint   = "appearance.appIconTint"
	pDockBadge = "appearance.dockBadge"
	pDockBnce  = "appearance.dockBounce"
	pDockProg  = "appearance.dockProgress"
	pFileIcon  = "appearance.fileIcon"
	pTrayStyle = "appearance.trayStyle"
	pTrayDyn   = "appearance.trayDyn"
	pTrayAnim  = "appearance.trayAnim"
	pTrayColor = "appearance.trayColorBadge"
	pTrayCount = "appearance.trayCount"
	pTrayClick = "appearance.trayClick"
)

var appStyles = []string{"经线", "纬线", "织纹", "格点", "斜纹", "经纬"}
var trayStyles = []string{"经线", "纬线", "格点", "斜纹", "织纹"}

// brandPreviewDock 是应用图标组的 Dock 预览条(即时反映偏好,不换真实 Dock 图标)。
func (a *App) brandPreviewDock(c *ui.Context) {
	t := a.tk
	style := a.prefs.Int(pAppStyle, 0)
	mode := a.prefs.Int(pAppMode, 0)
	ui.Row(c).Gap(10).Padding(4, 0).Children(func() {
		for i := 0; i < 3; i++ {
			col := []ui.Color{t.Ac, t.Info, t.Ok}[i%3]
			if mode == 2 { // 深色外观预览
				col = t.Tx1
			}
			ui.Column(c).Gap(4).AlignItems(ui.Center).Children(func() {
				ui.Row(c).Size(44, 44).Radius(10).Background(col).Border(1, t.Bd2).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
					ui.Text(c, appStyles[style%len(appStyles)][:1]).FontWeight(700).TextColor(t.OnAc)
				})
				ui.Text(c, []string{"Dock", "⌘Tab", "关于"}[i]).FontSize(theme.FsMicro).TextColor(t.Tx4)
			})
		}
	})
}

func (a *App) brandAppStylePick(c *ui.Context) {
	t := a.tk
	cur := a.prefs.Int(pAppStyle, 0)
	ui.Row(c).Gap(6).Wrap().Children(func() {
		for i, name := range appStyles {
			ch := kit.Chip(c, name, kit.ChipOpt{On: cur == i, Line: cur != i}).Key("appsty-" + itoa(i))
			if ch.Clicked() {
				a.prefs.Set(pAppStyle, i)
				setLog.Info("选择应用图标样式", "style", name)
				a.syncBrand()
			}
			_ = t
		}
	})
}

func (a *App) brandTrayStylePick(c *ui.Context) {
	cur := a.prefs.Int(pTrayStyle, 0)
	ui.Row(c).Gap(6).Wrap().Children(func() {
		for i, name := range trayStyles {
			ch := kit.Chip(c, name, kit.ChipOpt{On: cur == i, Line: cur != i}).Key("trsty-" + itoa(i))
			if ch.Clicked() {
				a.prefs.Set(pTrayStyle, i)
				setLog.Info("选择菜单栏图标样式", "style", name)
			}
		}
	})
}

func (a *App) brandTintPick(c *ui.Context) {
	t := a.tk
	cur := a.prefs.Int(pAppTint, 0)
	ui.Row(c).Gap(6).Children(func() {
		for i, label := range []string{"跟随主题", "宣纸", "墨夜", "雾霭", "湖烟", "杏笺", "暮岚"} {
			if i > 0 && i-1 < len(theme.Palettes) {
				_ = theme.Palettes[i-1]
			}
			ch := kit.Chip(c, label, kit.ChipOpt{On: cur == i, Line: cur != i}).Key("ictint-" + itoa(i))
			if cur == i {
				ch.Background(t.AcSoft).TextColor(t.AcTx)
			}
			if ch.Clicked() {
				if i == 0 {
					a.prefs.Reset(pAppTint)
				} else {
					a.prefs.Set(pAppTint, i)
				}
				a.syncBrand()
			}
		}
	})
}

// brandAppGroup / brandTrayGroup 供 settings_tabs 引用。
func brandAppGroup() setGroup {
	return setGroup{title: "应用图标", body: brandAppBody}
}

func brandTrayGroup() setGroup {
	return setGroup{title: "菜单栏图标", body: brandTrayBody}
}

func brandAppBody(a *App, c *ui.Context) {
	ui.Column(c).Padding(10, 12).Gap(6).Children(func() {
		a.brandPreviewDock(c)
	})
	a.settingsRow(c, setRow{key: pAppStyle, title: "图标样式", desc: "六选一;有对应 icns 时换 Dock,否则保持包内默认织纹", kind: kNone, right: func(a *App, c *ui.Context) {
		ui.Column(c).Gap(6).Children(func() {
			ui.Text(c, appStyles[a.prefs.Int(pAppStyle, 0)]).FontWeight(600).TextColor(a.tk.Tx2)
			a.brandAppStylePick(c)
		})
	}}, false)
	rows := []setRow{
		rSeg(pAppMode, "图标外观", "浅色 / 深色 / 透明 / 着色;「自动」随系统", 0, "自动", "浅色", "深色", "透明", "着色"),
		{key: pAppTint, title: "配色", desc: "跟随当前主题或固定色板", kind: kNone, right: func(a *App, c *ui.Context) { a.brandTintPick(c) }},
		rSeg(pDockBadge, "Dock 角标", "", 1, "关", "待处理数", "运行中数"),
		rSeg(pDockBnce, "需要确认时 Dock 弹跳", "", 1, "不弹", "一次", "直到处理"),
		rTog(pDockProg, "Dock 图标显示进度", "", false),
		rTog(pFileIcon, "同时更换访达里的图标", "", false),
	}
	for i, r := range rows {
		a.settingsRow(c, r, i == 0)
	}
}

func brandTrayBody(a *App, c *ui.Context) {
	a.settingsRow(c, rTog("general.tray", "在菜单栏显示图标", "关闭后可在本页重新打开", true), true)
	a.settingsRow(c, setRow{key: pTrayStyle, title: "图标样式", desc: "与应用图标同源的单色版", kind: kNone, right: func(a *App, c *ui.Context) {
		ui.Column(c).Gap(4).Children(func() {
			ui.Text(c, trayStyles[a.prefs.Int(pTrayStyle, 0)]).FontWeight(600)
			a.brandTrayStylePick(c)
		})
	}}, false)
	rows := []setRow{
		rTog(pTrayDyn, "随会话状态变化", "", true),
		rSeg(pTrayAnim, "运行中动画", "减弱动态时静止", 0, "织线流动", "呼吸", "无"),
		rTog(pTrayColor, "彩色徽标", "", true),
		rTog(pTrayCount, "在图标旁显示数字", "", true),
		rSeg(pTrayClick, "单击图标", "⌥ 单击执行另一个", 0, "弹出菜单", "打开窗口"),
	}
	for i, r := range rows {
		a.settingsRow(c, r, i == 0)
	}
}
