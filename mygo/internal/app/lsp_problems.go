package app

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/lsp"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// realProblems 是真实项目的「问题」面板:顶部分段(已打开 / 工作区)、筛选与重新检查,按文件分组诊断。
func (a *App) realProblems(c *ui.Context) {
	t := a.tk
	scope := a.probScope()
	var sessions []*lspSess
	for _, s := range a.lspH.sess {
		sessions = append(sessions, s)
	}
	sort.Slice(sessions, func(i, j int) bool { return sessions[i].spec.ID < sessions[j].spec.ID })

	files := a.sortedProblemFiles(scope)

	ui.Column(c).Grow(1).MinHeight(0).Children(func() {
		a.problemScopeBar(c)
		if scope == "ws" {
			a.problemWsChrome(c)
		}
		kit.VScroll(c).Key("probs").Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Padding(4, 8).Gap(2).Children(func() {
				for _, s := range sessions {
					if s.state == lspReady {
						continue
					}
					a.lspCard(c, s)
				}
				if len(files) == 0 && !a.anyWsScanning() {
					msg := "没有发现问题"
					if scope == "open" && len(a.lspH.sess) == 0 {
						msg = "打开一个源文件后,这里会显示语言服务器发现的问题"
					}
					errSrv, errMsg := a.wsScanError()
					_ = errSrv
					if scope == "ws" && errMsg == "" && !a.wsScanTruncated() {
						msg = "没有发现问题"
					}
					padY, padX := float32(6), float32(4)
					if scope == "ws" {
						padY, padX = 28, 28
					}
					if !(scope == "ws" && errMsg != "") { // 出错时只留红色错误条,不再写「没有发现问题」(设计稿错误态)
						ui.Text(c, msg).TextColor(t.Tx3).Padding(padY, padX).TextAlign(ui.Center)
					}
					for _, s := range sessions {
						if s.state == lspReady && scope == "open" {
							ui.Row(c).Gap(5).AlignItems(ui.Center).Padding(0, 4).Children(func() {
								kit.Ic(c, "okc", kit.S12).TextColor(t.Ok)
								ui.Text(c, s.spec.Name+"("+s.spec.Lang+")运行中").FontSize(theme.FsCap).TextColor(t.Tx4)
							})
						}
					}
					if scope == "ws" && a.anyWsScanning() {
						ui.Text(c, "扫描中… 已发现的诊断会边扫边出现").TextColor(t.Tx3).Padding(20).TextAlign(ui.Center)
					}
					return
				}
				if scope == "ws" && a.anyWsScanning() && len(files) == 0 {
					ui.Text(c, "扫描中… 已发现的诊断会边扫边出现").TextColor(t.Tx3).Padding(20).TextAlign(ui.Center)
				}
				diags := a.problemDiags(scope)
				for _, f := range files {
					f := f
					ds := append([]lsp.Diagnostic(nil), diags[f]...)
					sort.SliceStable(ds, func(i, j int) bool { return ds[i].Range.Start.Line < ds[j].Range.Start.Line })
					a.problemFileGroup(c, f, ds)
				}
			})
		})
		a.problemStatusLine(c, scope)
	})
}

func (a *App) problemScopeBar(c *ui.Context) {
	openN := a.problemCount("open")
	wsN := a.problemCount("ws")
	if a.wsDiagDemo != "" && a.proj == "" {
		wsN = map[string]int{"empty": 0, "scan": 0, "err": 0, "done": 12, "trunc": 2000}[a.wsDiagDemo]
		if wsN == 0 && a.wsDiagDemo == "trunc" {
			wsN = 2000
		}
	}
	scope := a.probScope()
	ui.Row(c).Height(theme.Px(28)).PaddingX(8).Gap(4).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
			kit.Chip(c, "问题 · 已打开", kit.ChipOpt{On: scope == "open"}).OnClick(func() {
				a.ide.probScope = "open"
				a.redraw()
			})
			if openN > 0 {
				kit.Cnt(c, itoa(openN), scope == "open")
			}
		})
		ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
			kit.Chip(c, "工作区", kit.ChipOpt{On: scope == "ws"}).OnClick(func() {
				a.ide.probScope = "ws"
				a.redraw()
			})
			kit.Cnt(c, itoa(wsN), scope == "ws")
		})
	})
}

func (a *App) problemWsChrome(c *ui.Context) {
	t := a.tk
	if a.anyWsScanning() {
		checked, total := a.wsScanProgress()
		pct := 0
		if total > 0 {
			pct = checked * 100 / total
		}
		ui.Row(c).Padding(4, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "已检查 "+itoa(checked)+" / "+itoa(total)+" 个文件").FontSize(theme.FsCap).TextColor(t.Tx3).Grow(1)
			ui.Text(c, itoa(pct)+"%").FontSize(theme.FsCap).TextColor(t.Tx3)
		})
		ui.Box(c).Height(3).Margin(0, 0, 4, 0).Children(func() { kit.IndetMeter(c) })
	}
	if srv, msg := a.wsScanError(); msg != "" {
		ui.Row(c).MinHeight(36).Padding(8, 12).Gap(8).Background(t.Err.Mix(t.BG, .88)).Children(func() {
			kit.Ic(c, "errc", kit.S14).TextColor(t.Err)
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, srv+" 没有响应").FontWeight(600)
				ui.Text(c, msg).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
			})
			kit.Btn(c, "重试", kit.BtnOpt{Small: true, Variant: kit.BtnPri}).OnClick(func() {
				for k := range a.lspH.wsScan {
					a.lspH.wsScan[k].Err = ""
				}
				a.wsDiagDemo = ""
				a.lspWorkspaceScan()
			})
		})
	}
	if a.wsScanTruncated() {
		limit := a.prefs.Int("lsp.workspaceDiagLimit", 2000)
		ui.Row(c).MinHeight(32).Padding(6, 12).Gap(8).Background(t.Warn.Mix(t.BG, .88)).Children(func() {
			kit.Ic(c, "alert", kit.S14).TextColor(t.Warn)
			ui.Text(c, "已达上限 "+itoa(limit)+" 个文件,只显示前 "+itoa(limit)+" 个").Grow(1).FontSize(theme.FsCap)
			kit.Btn(c, "调高上限…", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "sliders"}).OnClick(func() {
				a.jumpSettingRow("lsp", setRow{key: "lsp.workspaceDiagLimit", title: "工作区扫描文件上限"})
				a.openSettings("lsp")
			})
		})
	}
}

func (a *App) problemFileGroup(c *ui.Context, f string, ds []lsp.Diagnostic) {
	t := a.tk
	ui.Row(c).Key("pg-" + f).Height(theme.Px(24)).PaddingX(8).Gap(5).AlignItems(ui.Center).TextColor(t.Tx3).FontSize(theme.Px(10.5)).FontWeight(700).Children(func() {
		kit.Ic(c, "chevD", kit.S12)
		a.fileIc(c, f)
		ui.Text(c, f).Grow(1).SingleLine()
		kit.Cnt(c, itoa(len(ds)), false)
	})
	for i, d := range ds {
		d := d
		if !problemSeverityPass(d.Severity, a.ide.probErr) {
			continue
		}
		icon, col := "alert", t.Warn
		switch d.Severity {
		case 1:
			icon, col = "errc", t.Err
		case 3, 4:
			icon, col = "info", t.Info
		}
		prow := kit.RowEl(c, false).Key("pd-" + f + itoa(i)).Gap(6)
		prow.ContextMenu(func(m *ui.Menu) {
			a.problemMenu(m, f, d.Range.Start.Line+1, d.Message, func() string { return a.allProblemsText(a.sortedProblemFiles(a.probScope())) })
		})
		prow.Children(func() {
			kit.Ic(c, icon, kit.S12).TextColor(col)
			ui.Text(c, d.Message).Grow(1).SingleLine()
			if d.Source != "" {
				ui.Text(c, d.Source).FontSize(theme.FsCap).TextColor(t.Tx4).NoWrap()
			}
			ui.Text(c, "行 "+itoa(d.Range.Start.Line+1)).FontSize(theme.FsMicro).TextColor(t.Tx4).NoWrap()
		}).OnClick(func() { a.jumpToLSP(f, d.Range) })
	}
}

func (a *App) problemStatusLine(c *ui.Context, scope string) {
	t := a.tk
	if scope != "ws" {
		return
	}
	checked, total := a.wsScanProgress()
	src := a.wsScanSources()
	if src == "" && checked > 0 {
		src = "逐个打开扫描"
	}
	if src == "" {
		return
	}
	line := "共检查 " + itoa(max(checked, total)) + " 个文件 · 来源:" + src
	if a.anyWsScanning() {
		line = ""
	}
	if line == "" {
		return
	}
	ui.Text(c, line).FontSize(theme.FsCap).TextColor(t.Tx4).Padding(2, 12, 6, 12).SingleLine()
}

// lspCard 是问题面板顶部的服务器状态卡:没装 / 启动中 / 失败。
func (a *App) lspCard(c *ui.Context, s *lspSess) {
	t := a.tk
	card := ui.Row(c).Key("lc-"+s.spec.ID).MinHeight(36).Padding(6, 10).Gap(8).AlignItems(ui.Center).Radius(t.RSm()).Background(t.Fill1).Margin(0, 0, 4, 0)
	if s.uiAt.IsZero() {
		s.uiAt = time.Now()
	}
	card = kit.PopIn(card, s.uiAt)
	card.Children(func() {
		switch s.state {
		case lspStarting:
			kit.Spin(c, t.Tx3)
			ui.Text(c, s.spec.Name+"("+s.spec.Lang+")启动中…").TextColor(t.Tx2)
			return
		case lspMissing:
			kit.Ic(c, "alert", kit.S14).TextColor(t.Warn)
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, "未安装 "+s.spec.Name+"("+s.spec.Lang+" 语言服务)").FontWeight(500)
				ui.Text(c, s.spec.Install+" · "+s.spec.Hint).Font("monospace").FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
			})
			kit.Btn(c, "安装", kit.BtnOpt{Small: true, Variant: kit.BtnPri, Icon: "terminal"}).OnClick(func() { a.lspInstall(s) })
			if kit.Ib(c, "copy", kit.IbOpt{Small: true, Tip: "复制安装命令"}).Clicked() {
				a.copyText = s.spec.Install
				a.toast = "已复制安装命令"
			}
			if kit.Chip(c, "重新检测", kit.ChipOpt{Icon: "refresh"}).Clicked() {
				a.lspRetry(s)
			}
		default:
			kit.Ic(c, "errc", kit.S14).TextColor(t.Err)
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, s.spec.Name+" 没有运行").FontWeight(500)
				ui.Text(c, s.err).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
			})
			kit.Btn(c, "重试", kit.BtnOpt{Small: true, Variant: kit.BtnSoft, Icon: "refresh"}).OnClick(func() { a.lspRetry(s) })
		}
	})
}

// lspRelFile 把项目相对路径换成绝对路径(测试用)。
func (a *App) lspAbs(rel string) string { return filepath.Join(a.proj, rel) }

// allProblemsText 把全部诊断拼成「文件:行 消息」的多行文本(右键「复制全部」)。
func (a *App) allProblemsText(files []string) string {
	var sb strings.Builder
	diags := a.problemDiags(a.probScope())
	for _, f := range files {
		for _, d := range diags[f] {
			sb.WriteString(f + ":" + itoa(d.Range.Start.Line+1) + " " + d.Message + "\n")
		}
	}
	return strings.TrimSpace(sb.String())
}
