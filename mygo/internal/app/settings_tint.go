package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 设置 → 主题与布局 → 背景与表面(设计稿 css/tint.css + settings.js):背景样式 / 柔光强度 / 表面色调 / 自定义色相与浓淡 / 层次预览。
// 偏好键与取值见 settings.go 的 kBgStyle … kTintK;令牌推导见 theme/tint.go;窗口底光晕见 shell.go 与 settings.go 的 bgGlow。
//
// @author ixxxxoooo

// showIf 给一行加上显示条件(见 setRow.show)。
func showIf(r setRow, f func(a *App) bool) setRow { r.show = f; return r }

func glowOn(a *App) bool     { return a.prefs.Int(kBgStyle, 0) == 2 }
func tintCustom(a *App) bool { return a.prefs.Int(kTint, 0) == theme.TintCustom }

// bgSurfaceRows 是「背景与表面」分组的行。
func bgSurfaceRows() []setRow {
	return []setRow{
		rSeg(kBgStyle, "背景样式", "跟随主题:现状(浮岛布局带一抹强调色光晕,其余纯色);纯色:窗口底是一个平色,没有任何渐变与光晕;柔光:任何主题与布局都在左上角铺一抹强调色光晕。流光主题的壁纸同样适用", 0, "跟随主题", "纯色", "柔光"),
		showIf(rSeg(kGlow, "柔光强度", "强调色在光晕中心的占比:淡 5% / 标准 9% / 浓 14%;只在「柔光」下可调", 1, "淡", "标准", "浓"), glowOn),
		{key: kTint, title: "表面色调", kind: kNone, def: 0,
			desc:  "给窗口底、侧栏、面板、卡片、凹槽这几层中性灰一个色相;各层之间的明暗差不变,所以不是整体刷成一个色,文字对比度也不变。选一个预设,或在「自定义」里自己调;不影响成功 / 报错 / 警告与语法高亮的颜色。选了预设后,强调色选「沿用主题」会换成该色调的推荐色(仍可单独改)",
			right: func(a *App, c *ui.Context) { tintSwatches(a, c) }},
		showIf(rStp(kTintHue, "色相", "自定义色调的色相,0–355°(0 红 · 85 黄褐 · 150 绿 · 255 蓝 · 320 紫);只在选了「自定义」时出现", 85, 0, 355, 5, "°"), tintCustom),
		showIf(rStp(kTintK, "浓淡", "自定义色调的饱和度,0 = 纯中性灰,100 = 最浓(仍是很淡的灰,不会变成彩色底);只在选了「自定义」时出现", 40, 0, 100, 5, ""), tintCustom),
		rPreview("tint"),
	}
}

// tintSwatches 是表面色调的圆点行:跟随主题(对角分半)+ 7 个预设 + 自定义(色相环)。
func tintSwatches(a *App, c *ui.Context) {
	t := a.tk
	cur := a.prefs.Int(kTint, 0)
	ui.Row(c).Gap(8).Shrink(0).Children(func() {
		for i := range theme.TintPresets {
			p := theme.TintPresets[i]
			name := p.Name
			b := ui.ButtonBase(c).Key("tint-"+itoa(i)).Size(18, 18).Radius(9).Cursor(ui.CursorPointer).Shrink(0).Tooltip(name).
				Transition(ui.ElementTransition{Size: true, Duration: kit.DurBase, Ease: kit.EaseSpring})
			switch {
			case i == 0: // 跟随主题:窗口底 / 卡片色各半
				b.LinearGradient(ui.LinearGradient{Angle: 135, Stops: []ui.GradientStop{{Color: t.Chrome, At: 0}, {Color: t.Chrome, At: .5}, {Color: t.Raised, At: .5}, {Color: t.Raised, At: 1}}})
			case i == theme.TintCustom:
				b.ConicGradient(ui.ConicGradient{Stops: theme.TintHueRing(12)})
			default:
				b.Background(theme.TintSwatch(p.Hue, p.K))
			}
			if i == cur {
				b.Shadow(0, 0, 0, 2, t.Raised).Shadow(0, 0, 0, 3.5, t.Tx1)
			} else {
				b.Border(1, t.Bd2)
			}
			if b.Clicked() {
				if i == 0 {
					a.prefs.Reset(kTint)
				} else {
					a.prefs.Set(kTint, i)
				}
				a.applyPrefs()
			}
		}
	})
}

// pvTint 是层次预览:四条色带(窗口底 / 面板 / 卡片 / 凹槽),看得出「不是整体一个色」。
func (a *App) pvTint(c *ui.Context) {
	t := a.tk
	row := ui.Row(c).Key("set-pv-tint").Margin(10, 12).Height(54).Radius(t.RMd()).Border(1, t.Bd2).Clip()
	row.Children(func() {
		for _, s := range []struct {
			name string
			col  ui.Color
		}{{"窗口底", t.Chrome}, {"面板", t.BG}, {"卡片", t.Raised}, {"凹槽", t.Sunken}} {
			ui.Column(c).Grow(1).Basis(0).Background(s.col).Justify(ui.End).Padding(6, 8).Children(func() {
				ui.Text(c, s.name).FontSize(theme.FsMicro).TextColor(t.Tx3)
			})
		}
	})
}
