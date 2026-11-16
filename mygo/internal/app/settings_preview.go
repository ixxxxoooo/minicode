package app

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diff"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

const shimmerPreviewTick = 33 * time.Millisecond

// settingsPreview 按 id 绘制设置页外观类实时预览(设计稿 js/settings-pv.js)。
func (a *App) settingsPreview(c *ui.Context, id string) {
	switch id {
	case "typo":
		a.pvTypo(c)
	case "width":
		a.pvWidth(c)
	case "tools":
		a.pvTools(c)
	case "shimmer":
		a.pvShimmer(c)
	case "spinner":
		a.pvSpinner(c)
	case "wait":
		a.pvWait(c)
	}
}

// setPvShell 是预览凹槽:右上角「预览」标,底色与内容区 BG 一致。
func (a *App) setPvShell(c *ui.Context, key string, body func()) {
	t := a.tk
	box := ui.Column(c).Key("set-pv-"+key).Margin(8, 12, 10, 12).Padding(12, 14).Background(t.BG).Radius(t.RMd()).Border(1, t.Bd1).Clip()
	box.Children(func() {
		ui.Text(c, "预览").Absolute().Top(6).Right(9).FontSize(theme.FsMicro).TextColor(t.Tx4).LetterSpacing(.04)
		body()
	})
}

func (a *App) pvNote(c *ui.Context, s string) {
	t := a.tk
	ui.Text(c, s).FontSize(theme.FsCap).TextColor(t.Tx3).LineHeight(1.45).Margin(8, 0, 0, 0)
}

func pvOpacity(active bool) float32 {
	if active {
		return 1
	}
	return 0.38
}

func pvRound1(v float32) string {
	return fmt.Sprintf("%.1f", math.Round(float64(v)*10)/10)
}

// ───────── 1) 字号与会话片段 ─────────

func (a *App) pvTypo(c *ui.Context) {
	t := a.tk
	u := &core.User{ID: "pv-typo", Text: "路由匹配对大小写敏感:`/Users` 和 `/users` 应该命中同一个处理器。@{router.rs}"}
	md := "问题出在路由表用原始路径作键。最小改动是抽一个 `normalize`,注册与匹配共用,顺带处理末尾斜杠。"
	a.setPvShell(c, "typo", func() {
		ui.Column(c).Gap(4).Padding(2, 0).Children(func() {
			ui.ButtonBase(c).Background(t.Raised.Over(t.BG)).Radius(t.RMd()).Padding(7, 11).AlignItems(ui.Stretch).Column().Children(func() {
				a.userMsgBody(c, u, false)
			})
			markdown.View(c, padCode(md), a.mdOptions())
			ui.Column(c).Gap(1).Children(func() {
				a.stepRow(c, step{id: "pv-read", icon: "fileText", verb: "读取", obj: "router.rs", meta: "src/ · 212 行", path: "src/router.rs", fam: famExplore}, nil)
				a.stepRow(c, step{id: "pv-edit", icon: "edit", verb: "编辑", obj: "router.rs", meta: "src/", add: 4, del: 1, path: "src/router.rs", fam: famEdit, diffAs: true}, nil)
				a.stepRow(c, step{id: "pv-run", icon: "terminal", verb: "运行", obj: "cargo test router", right: "12 passed", fam: famExec}, nil)
			})
		})
		a.pvNote(c, fmt.Sprintf("界面 %s · 阅读 %s · 代码 %s · 行距 %s",
			pvRound1(theme.FsBody), pvRound1(a.FlowFontSize()), pvRound1(a.FlowMonoSize()), pvRound1(a.FlowReadLH())))
	})
}

// ───────── 2) 内容宽度示意 ─────────

var pvFlowW = []float32{560, 640, theme.WCol, 860, 1040, 1400}
var pvSetW = []float32{620, 760, 900, 1080, 1400}

func (a *App) pvWidth(c *ui.Context) {
	fi := a.flowWidthIndex()
	si := clampInt(a.prefs.Int(pSetWidth, 1), 0, setWidthFull)
	flowFull := fi == flowWidthFull
	setFull := si == setWidthFull
	fp := float32(100)
	if !flowFull {
		fp = min(100, float32(int(pvFlowW[fi]/900*100+0.5)))
	}
	sp := float32(100)
	if !setFull {
		sp = min(100, float32(int(pvSetW[si]/1000*100+0.5)))
	}
	flowLbl := "占满"
	if !flowFull {
		flowLbl = fmt.Sprintf("%dpx", int(pvFlowW[fi]))
	}
	setLbl := "占满"
	if !setFull {
		setLbl = fmt.Sprintf("%dpx", int(pvSetW[si]))
	}
	a.setPvShell(c, "width", func() {
		ui.Row(c).Gap(10).Children(func() {
			a.pvWidthWin(c, "会话", fp, flowLbl, true)
			a.pvWidthWin(c, "设置页", sp, setLbl, false)
		})
		a.pvNote(c, "窗口较窄时内容列自动收窄,不会出现横向滚动条。")
	})
}

func (a *App) pvWidthWin(c *ui.Context, label string, pct float32, val string, flow bool) {
	t := a.tk
	ui.Column(c).Grow(1).MinWidth(0).Children(func() {
		ui.Row(c).Margin(0, 0, 4, 0).AlignItems(ui.Center).FontSize(theme.FsCap).TextColor(t.Tx3).Children(func() {
			ui.Text(c, label)
			ui.Spacer(c).Grow(1)
			ui.Text(c, val).TextColor(t.Tx2).FontFeatures("tnum")
		})
		ui.Box(c).Height(64).Radius(t.RSm()).Background(t.Sunken).Border(1, t.Bd2).Justify(ui.Center).Padding(6, 0).Clip().Children(func() {
			col := ui.Column(c).WidthPercent(pct).MaxWidthPercent(100).Gap(4).Children(func() {
				if flow {
					ui.Box(c).Height(7).Radius(3).Background(t.Fill3)
					ui.Box(c).Height(7).Radius(3).Background(t.Fill3)
					ui.Box(c).Height(7).WidthPercent(62).Radius(3).Background(t.Fill3)
					ui.Box(c).Height(7).Radius(3).Background(t.Fill3)
					ui.Box(c).Height(12).Radius(6).Background(t.Raised).Border(1, t.Bd3).Margin(4, 0, 0, 0)
				} else {
					ui.Box(c).Height(9).WidthPercent(40).Radius(3).Background(t.Tx4)
					ui.Box(c).Height(7).Radius(3).Background(t.Fill3)
					ui.Box(c).Height(7).Radius(3).Background(t.Fill3)
					ui.Box(c).Height(7).WidthPercent(62).Radius(3).Background(t.Fill3)
				}
			})
			_ = col
		})
	})
}

// ───────── 3) 工具调用详细程度 ─────────

var pvToolsCap = []string{
	"相邻的工具调用合并成一行摘要",
	"只合并连续的读取 / 搜索;编辑与命令各占一行,点开看详情",
	"每个工具调用一行,编辑的差异与命令输出直接显示在行下",
}

func (a *App) pvTools(c *ui.Context) {
	t := a.tk
	lv := a.ToolDetailLevel()
	a.setPvShell(c, "tools", func() {
		ui.Column(c).Gap(1).Children(func() {
			switch lv {
			case 0:
				ui.Row(c).Height(30).Padding(0, 10).Gap(8).Radius(t.RMd()).Background(t.Raised).Border(1, t.Bd1).AlignItems(ui.Center).TextColor(t.Tx2).Children(func() {
					kit.Ic(c, "layers", kit.S12).TextColor(t.Tx3)
					ui.Text(c, "编辑了 2 个文件、运行了 2 条命令").Grow(1).MinWidth(0).SingleLine()
					kit.Ic(c, "chevR", kit.S12).TextColor(t.Tx4)
				})
			case 2:
				a.stepRow(c, step{id: "pv-t-edit", icon: "edit", verb: "编辑", obj: "src/router.rs", add: 4, del: 1, fam: famEdit, hasBody: true, open: true, diffAs: true}, func() {
					a.pvMiniDiff(c)
				})
				a.stepRow(c, step{id: "pv-t-run", icon: "terminal", verb: "运行", obj: "cargo test router", fam: famExec, hasBody: true, open: true}, func() {
					a.pvMiniTerm(c)
				})
			default:
				a.stepRow(c, step{id: "pv-t-e1", icon: "edit", verb: "编辑", obj: "src/router.rs", add: 4, del: 1, fam: famEdit}, nil)
				a.stepRow(c, step{id: "pv-t-e2", icon: "edit", verb: "编辑", obj: "README.md", add: 5, del: 1, fam: famEdit}, nil)
				a.stepRow(c, step{id: "pv-t-r1", icon: "terminal", verb: "运行", obj: "cargo test router", fam: famExec}, nil)
				a.stepRow(c, step{id: "pv-t-r2", icon: "terminal", verb: "运行", obj: "cargo check", fam: famExec}, nil)
			}
		})
		if lv >= 0 && lv < len(pvToolsCap) {
			a.pvNote(c, pvToolsCap[lv])
		}
	})
}

func (a *App) pvMiniDiff(c *ui.Context) {
	t := a.tk
	d := &core.DiffBlock{
		Path: "src/router.rs",
		Old:  "    path.trim_end_matches('/')\n    registry.get(path)\n",
		New:  "    let key = normalize(path);\n    registry.get(&key)\n",
		Line: 41,
	}
	ls := diff.Lines(d.Old, d.New, d.Line, d.Line)
	a.cardM(c, nil, nil, func() {
		ui.Column(c).Padding(3, 0).Children(func() {
			for i, l := range ls {
				if i >= 2 {
					break
				}
				a.diffLine(c, i, l)
			}
		})
	})
	_ = t
}

func (a *App) pvMiniTerm(c *ui.Context) {
	t := a.tk
	a.cardM(c, nil, nil, func() {
		ui.Row(c).Padding(4, 8).Font("monospace").FontSize(a.FlowMonoSize()).TextColor(t.Tx3).Children(func() {
			ui.Text(c, "test result: ")
			ui.Text(c, "ok").TextColor(t.Ok)
			ui.Text(c, ". 14 passed")
		})
	})
}

// ───────── 4) 流光样式 ─────────

func (a *App) pvShimmer(c *ui.Context) {
	t := a.tk
	cur := a.prefs.Int(pShimmer, 0)
	labels := []string{"流光", "呼吸", "无"}
	a.setPvShell(c, "shimmer", func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			for i := 0; i < 3; i++ {
				i := i
				ui.Column(c).Gap(4).Opacity(pvOpacity(i == cur)).Children(func() {
					kit.RunTextMode(c, "正在分析路由表…", t.Tx2, i, func(e ui.Element) ui.Element {
						return e.FontSize(theme.FsCap)
					})
					ui.Text(c, labels[i]).FontSize(theme.FsCap).TextColor(t.Tx4)
				})
			}
			if kit.MotionOn() {
				a.frameIn(shimmerPreviewTick)
			}
		})
	})
}

// ───────── 5) 等待字符动画 ─────────

func (a *App) pvSpinner(c *ui.Context) {
	t := a.tk
	idx := a.prefs.Int(pSpinner, 2)
	if idx < 0 || idx >= len(spinnerDefs) {
		idx = 2
	}
	d := spinnerDefs[idx]
	if kit.MotionOn() {
		a.frameIn(glyphFrameMs * time.Millisecond)
	}
	a.setPvShell(c, "spinner", func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, glyphFrame(d)).FontSize(theme.Px(14)).Width(14).TextAlign(ui.Center).TextColor(t.Ac).SingleLine().Shrink(0)
			ui.Text(c, spinnerVerb(d, "pv-spin")+"…").TextColor(t.Tx2).SingleLine().Shrink(0)
			ui.Text(c, "12 秒 · ↓ 1.2k tokens · 按 Esc 中断").FontSize(theme.FsCap).TextColor(t.Tx4).FontFeatures("tnum").SingleLine().Shrink(1)
		})
		a.pvNote(c, d.name+":状态行与回合进行中的字符。")
	})
}

// ───────── 6) 等待指示样式 ─────────

var pvWaitNames = []string{"沙漏", "圆环", "点阵", "脉冲"}

func (a *App) pvWait(c *ui.Context) {
	t := a.tk
	cur := a.prefs.Int(pWaitAnim, 0)
	if kit.MotionOn() {
		a.frameIn(33 * time.Millisecond)
	}
	a.setPvShell(c, "wait", func() {
		ui.Row(c).Gap(10).Children(func() {
			for i := 0; i < 4; i++ {
				i := i
				ui.Column(c).Gap(4).Grow(1).MinWidth(0).Opacity(pvOpacity(i == cur)).Children(func() {
					a.pvWaitStep(c, i)
					ui.Text(c, pvWaitNames[i]).FontSize(theme.FsCap).TextColor(t.Tx4)
				})
			}
		})
	})
}

func (a *App) pvWaitStep(c *ui.Context, style int) {
	t := a.tk
	tone := t.ToolExec
	ui.Row(c).MinHeight(24).Padding(0, 8).Gap(6).Radius(t.RSm()).AlignItems(ui.Center).TextColor(t.Tx2).Children(func() {
		ui.Row(c).Size(16, 16).Justify(ui.Center).AlignItems(ui.Center).Shrink(0).Children(func() {
			kit.SpinStyle(c, tone, style)
		})
		kit.RunTextS(c, "运行", t.Tx3, func(e ui.Element) ui.Element {
			return e.TextColor(t.Tx3).Shrink(0).SingleLine()
		})
		ui.Text(c, "cargo test").FontWeight(500).TextColor(t.Tx1).SingleLine().Shrink(1)
	})
}
