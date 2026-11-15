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

// realProblems 是真实项目的「问题」面板:顶部是各语言服务器的状态(没装 → 一键安装 / 复制命令,
// 失败 → 重新检测),下面是按文件分组的诊断,点一条跳到位置。
func (a *App) realProblems(c *ui.Context) {
	t := a.tk
	var sessions []*lspSess
	for _, s := range a.lspH.sess {
		sessions = append(sessions, s)
	}
	sort.Slice(sessions, func(i, j int) bool { return sessions[i].spec.ID < sessions[j].spec.ID })

	files := make([]string, 0, len(a.lspH.diags))
	for f := range a.lspH.diags {
		files = append(files, f)
	}
	sort.Strings(files)

	ui.Scroll(c).Key("probs").Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Padding(4, 8).Gap(2).Children(func() {
			for _, s := range sessions {
				s := s
				if s.state == lspReady {
					continue
				}
				a.lspCard(c, s)
			}
			if len(files) == 0 {
				msg := "没有检测到问题"
				if len(a.lspH.sess) == 0 {
					msg = "打开一个源文件后,这里会显示语言服务器发现的问题"
				}
				ui.Text(c, msg).TextColor(t.Tx3).Padding(6, 4)
				for _, s := range sessions {
					if s.state == lspReady {
						ui.Row(c).Gap(5).AlignItems(ui.Center).Padding(0, 4).Children(func() {
							kit.Ic(c, "okc", kit.S12).TextColor(t.Ok)
							ui.Text(c, s.spec.Name+"("+s.spec.Lang+")运行中").FontSize(theme.FsCap).TextColor(t.Tx4)
						})
					}
				}
				return
			}
			for _, f := range files {
				f := f
				ds := append([]lsp.Diagnostic(nil), a.lspH.diags[f]...)
				sort.SliceStable(ds, func(i, j int) bool { return ds[i].Range.Start.Line < ds[j].Range.Start.Line })
				ui.Row(c).Key("pg-" + f).Height(24).PaddingX(8).Gap(5).AlignItems(ui.Center).TextColor(t.Tx3).FontSize(10.5).FontWeight(700).Children(func() {
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
						a.problemMenu(m, f, d.Range.Start.Line+1, d.Message, func() string { return a.allProblemsText(files) })
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
		})
	})
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
	for _, f := range files {
		for _, d := range a.lspH.diags[f] {
			sb.WriteString(f + ":" + itoa(d.Range.Start.Line+1) + " " + d.Message + "\n")
		}
	}
	return strings.TrimSpace(sb.String())
}
