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

// 样式名与顺序以设计稿 brand.js::STYLES 为准(下标 = 偏好值,也是素材文件名里的编号)。
var appStyles = []string{"织纹", "经纬", "提示符", "棋盘", "行", "花括号"}
var trayStyles = []string{"织纹", "提示符", "棋盘", "行", "花括号"}

// brandTintPick 「配色」:第一枚「跟随主题」(图标随当前主题变色),其后六枚固定为某一套主题的配色,
// 每枚都是该配色下当前样式的图标小预览(设计稿 brand.js::tintPick)。
func (a *App) brandTintPick(c *ui.Context) {
	t := a.tk
	cur := a.prefs.Int(pAppTint, 0)
	style := a.prefs.Int(pAppStyle, 0) % len(appStyles)
	set := func(i int) {
		if i == 0 {
			a.prefs.Reset(pAppTint)
		} else {
			a.prefs.Set(pAppTint, i)
		}
		setLog.Info("选择应用图标配色", "tint", i)
		a.syncBrand()
	}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
		ch := kit.Chip(c, "跟随主题", kit.ChipOpt{On: cur == 0, Line: cur != 0}).Key("ictint-0")
		if cur == 0 {
			ch.Background(t.AcSoft).TextColor(t.AcTx)
		}
		if ch.Clicked() {
			set(0)
		}
		for i, p := range theme.Palettes {
			i, p := i+1, p
			on := cur == i
			b := ui.ButtonBase(c).Key("ictint-"+itoa(i)).Size(40, 40).Radius(t.RMd()).Cursor(ui.CursorPointer).
				Shrink(0).Tooltip(p.Name)
			switch {
			case on:
				b.Background(t.AcSoft).Border(1.5, t.Ac)
			case b.Hovered():
				b.Background(t.Fill1).Border(1, t.Bd2)
			default:
				b.Border(1, t.Bd1)
			}
			b.Justify(ui.Center).AlignItems(ui.Center).Children(func() {
				a.brandAppImg(c, a.appIconNameFor(style, iconPalOf(p.ID), p.Dark), 28)
			})
			if b.Clicked() {
				set(i)
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
	a.settingsRow(c, setRow{key: pAppStyle, def: 0, title: "图标样式", desc: "六选一;立即换 Dock 图标", kind: kNone, right: func(a *App, c *ui.Context) {
		ui.Column(c).Gap(6).Children(func() {
			ui.Text(c, appStyles[a.prefs.Int(pAppStyle, 0)]).FontWeight(600).TextColor(a.tk.Tx2)
			a.brandAppStylePick(c)
		})
	}}, false)
	rows := []setRow{
		rSeg(pAppMode, "图标外观", "浅色 / 深色 / 透明 / 着色;「自动」随当前主题深浅", 0, "自动", "浅色", "深色", "透明", "着色"),
		{key: pAppTint, title: "配色", desc: "默认跟随当前主题变色,也可固定为某一套主题的配色", kind: kNone, right: func(a *App, c *ui.Context) { a.brandTintPick(c) }},
		rSeg(pDockBadge, "Dock 角标", "", 1, "关", "待处理数", "运行中数"),
		rSeg(pDockBnce, "需要确认时 Dock 弹跳", "", 1, "不弹", "一次", "直到处理"),
		rTog(pDockProg, "Dock 图标显示进度", "", false),
		rTog(pFileIcon, "同时更换访达里的图标", "把当前样式设为 MiniCode.app 在访达里的图标;关闭即还原。需要应用所在位置可写,开发版每次重新打包会还原", false),
	}
	for i, r := range rows {
		a.settingsRow(c, r, i == 0)
	}
}

func brandTrayBody(a *App, c *ui.Context) {
	a.settingsRow(c, rTog("general.tray", "在菜单栏显示图标", "关闭后可在本页重新打开", true), true)
	a.settingsRow(c, setRow{key: pTrayStyle, def: 0, title: "图标样式", desc: "五选一,单色模板图,随菜单栏深浅自动变色", kind: kNone, right: func(a *App, c *ui.Context) {
		ui.Column(c).Gap(4).Children(func() {
			ui.Text(c, trayStyles[a.prefs.Int(pTrayStyle, 0)]).FontWeight(600)
			a.brandTrayStylePick(c)
		})
	}}, false)
	ui.Column(c).Padding(4, 12, 8, 12).Gap(6).Children(func() {
		ui.Text(c, "各状态预览(上浅色菜单栏 / 下深色菜单栏)").FontSize(theme.FsCap).TextColor(a.tk.Tx3)
		a.brandTrayStates(c)
	})
	rows := []setRow{
		rTog(pTrayDyn, "随会话状态变化", "", true),
		rSeg(pTrayAnim, "运行中动画", "系统「减弱动态」时静止", 0, "织线流动", "呼吸", "无"),
		rTog(pTrayColor, "彩色徽标", "待确认 / 出错 / 有更新 / 已完成用彩色圆点;「增强对比度」下自动关闭", true),
		rTog(pTrayCount, "在图标旁显示数字", "", true),
		rSeg(pTrayClick, "单击图标", "⌥ 单击执行另一个", 0, "弹出菜单", "打开窗口"),
	}
	for i, r := range rows {
		a.settingsRow(c, r, i == 0)
	}
}
