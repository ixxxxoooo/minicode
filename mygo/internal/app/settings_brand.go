package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/prefs"
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
	// pTrayStyleLegacy 菜单栏图标曾有独立的样式偏好;现在与应用图标样式合一(pAppStyle),
	// 这个键只在迁移时清掉,不再读取。
	pTrayStyleLegacy = "appearance.trayStyle"
	pTrayDyn         = "appearance.trayDyn"
	pTrayAnim        = "appearance.trayAnim"
	pTrayColor       = "appearance.trayColorBadge"
	pTrayCount       = "appearance.trayCount"
	pTrayClick       = "appearance.trayClick"
	// pIconStyleRev 图标样式编号的版本:2 = 已删掉「织纹」(提示符成了 0);3 = 菜单栏图标样式并入应用图标样式。
	pIconStyleRev = "appearance.iconStyleRev"
)

// 样式名与顺序以设计稿 brand.js::STYLES 为准(下标 = 偏好值,也是素材文件名里的编号)。
// 默认(下标 0)是提示符;应用图标与菜单栏图标共用这一份样式:选一次,Dock / ⌘Tab / 访达 / 菜单栏同时换成同一种。
var appStyles = []string{"提示符", "经纬", "棋盘", "行", "花括号"}

// migrateIconStyles 迁移图标样式偏好(只做一次,以 pIconStyleRev 为记号):
//   - 版本 < 2(删掉织纹之前):应用图标 旧 织纹 0 / 经纬 1 / 提示符 2 / 棋盘 3 / 行 4 / 花括号 5 → 新 0 / 1 / 0 / 2 / 3 / 4;
//   - 版本 < 3:菜单栏图标不再有独立样式,清掉旧的 appearance.trayStyle,一律跟随应用图标样式。
//
// 从没选过的人(键不存在)本来就跟随默认,现在自然变成提示符。
func migrateIconStyles(p *prefs.Store) {
	rev := 0
	if p != nil {
		rev = p.Int(pIconStyleRev, 0)
	}
	if p == nil || rev >= 3 {
		return
	}
	conv := func(key string, table []int) {
		if !p.Has(key) {
			return
		}
		old := p.Int(key, 0)
		n := 0
		if old >= 0 && old < len(table) {
			n = table[old]
		}
		if n == 0 {
			p.Reset(key)
		} else {
			p.Set(key, n)
		}
		setLog.Info("迁移图标样式偏好", "key", key, "old", old, "new", n)
	}
	if rev < 2 {
		conv(pAppStyle, []int{0, 1, 0, 2, 3, 4})
	}
	if p.Has(pTrayStyleLegacy) {
		setLog.Info("菜单栏图标样式并入应用图标样式,清掉旧的独立偏好", "old", p.Int(pTrayStyleLegacy, 0))
		p.Reset(pTrayStyleLegacy)
	}
	p.Set(pIconStyleRev, 3)
}

// brandTintPick 「配色」:第一枚「跟随主题」(图标随当前主题变色),其后六枚固定为某一套主题的配色,
// 每枚都是该配色下当前样式的图标小预览(设计稿 brand.js::tintPick)。
func (a *App) brandTintPick(c *ui.Context) {
	t := a.tk
	cur := a.prefs.Int(pAppTint, 0)
	style := a.iconStyleIndex()
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
		a.brandLivePreview(c)
		a.brandPreviewDock(c)
	})
	a.settingsRow(c, setRow{key: pAppStyle, def: 0, title: "图标样式", desc: "五选一;应用图标与菜单栏图标同时换成同一种,立即生效。鼠标放在卡片上能看到菜单栏图标动起来的样子", kind: kNone, right: func(a *App, c *ui.Context) {
		a.brandAppStylePick(c)
	}}, false)
	rows := []setRow{
		rSeg(pAppMode, "图标外观", "浅色 / 深色 / 透明 / 着色;「自动」随当前主题深浅", 0, "自动", "浅色", "深色", "透明", "着色"),
		{key: pAppTint, title: "配色", desc: "默认跟随当前主题变色,也可固定为某一套主题的配色", kind: kNone, right: func(a *App, c *ui.Context) { a.brandTintPick(c) }},
		rSeg(pDockBadge, "Dock 角标", "", 1, "关", "待处理数", "运行中数"),
		rSeg(pDockBnce, "需要确认时 Dock 弹跳", "", 1, "不弹", "一次", "直到处理"),
		rTog(pDockProg, "Dock 图标显示进度", "", false),
		rTog(pFileIcon, "同时更换访达里的图标", "把当前样式设为 MiniCode.app 在访达里的图标(访达、启动台、关于本机);关闭即还原。需要应用所在位置可写;应用更新后第一次启动会自动重新套用", false),
	}
	for i, r := range rows {
		a.settingsRow(c, r, i == 0)
	}
	a.brandStatusRow(c)
}

// brandStatusRow「当前生效」:把 Dock / 访达 / 菜单栏三处实际套用的结果写出来,外加一键「重新应用」。
// 设置本身是即时生效的(改完立刻同步);这一行是兜底与排查入口——哪一处没成功、为什么,一眼可见。
func (a *App) brandStatusRow(c *ui.Context) {
	a.settingsRow(c, setRow{title: "当前生效", desc: "改完设置会立刻套用,不需要重启;若某一处没变,点右边「重新应用」", kind: kNone, right: func(a *App, c *ui.Context) {
		t := a.tk
		ui.Column(c).Gap(4).AlignItems(ui.End).Children(func() {
			line := func(label, val string) {
				col := t.Tx3
				if len(val) >= 2 && val[:2] == "失败" {
					col = t.Err
				}
				ui.Row(c).Gap(6).Children(func() {
					ui.Text(c, label).FontSize(theme.FsCap).TextColor(t.Tx4)
					ui.Text(c, val).FontSize(theme.FsCap).TextColor(col)
				})
			}
			line("Dock", orDash(a.brand.dockState, "待应用"))
			line("访达", orDash(a.brand.finderState, finderOff))
			if a.brand.tray != nil {
				line("菜单栏", "已显示 · "+appStyles[a.iconStyleIndex()])
			} else {
				line("菜单栏", "未显示(见下方「在菜单栏显示图标」)")
			}
			kit.Btn(c, "重新应用", kit.BtnOpt{Icon: "refresh"}).Key("brand-reapply").OnClick(a.reapplyBrand)
		})
	}}, false)
}

func orDash(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func brandTrayBody(a *App, c *ui.Context) {
	a.settingsRow(c, rTog("general.tray", "在菜单栏显示图标", "关闭后可在本页重新打开", true), true)
	a.settingsRow(c, setRow{title: "图标样式", desc: "与应用图标是同一个样式(单色模板图,随菜单栏深浅自动变色);要换样式请在上方「应用图标」里选,两处同时变", kind: kNone, right: func(a *App, c *ui.Context) {
		ui.Text(c, appStyles[a.iconStyleIndex()]+" · 同应用图标").FontWeight(500).TextColor(a.tk.Tx2)
	}}, false)
	ui.Column(c).Padding(4, 12, 8, 12).Gap(6).Children(func() {
		ui.Text(c, "各状态预览(上浅色菜单栏 / 下深色菜单栏;运行中那一格会动)").FontSize(theme.FsCap).TextColor(a.tk.Tx3)
		a.brandTrayStates(c)
	})
	rows := []setRow{
		rTog(pTrayDyn, "随会话状态变化", "", true),
		rSeg(pTrayAnim, "运行中动画", "系统「减弱动态」时静止", 0, "逐段点亮", "呼吸", "无"),
		rTog(pTrayColor, "彩色徽标", "待确认 / 出错 / 有更新 / 已完成用彩色圆点;「增强对比度」下自动关闭", true),
		rTog(pTrayCount, "在图标旁显示数字", "", true),
		rSeg(pTrayClick, "单击图标", "「弹出菜单」:单击弹出菜单;「打开窗口」:单击打开主窗口,右键弹出菜单", 0, "弹出菜单", "打开窗口"),
	}
	for i, r := range rows {
		a.settingsRow(c, r, i == 0)
	}
}
