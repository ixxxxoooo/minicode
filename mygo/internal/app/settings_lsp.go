package app

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/lsp"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

type lspSettingRow struct {
	ID      string
	Name    string
	Lang    string
	Exts    string
	Path    string
	Install string
	OK      bool
	Enabled bool
}

func (a *App) lspServerPrefKey(id string) string { return "lsp.server." + id }

func (a *App) scanLspSettings() {
	if a.demoWs() {
		var rows []lspSettingRow
		for _, r := range demo.LSPDemoRows() {
			rows = append(rows, lspSettingRow{
				ID: r.ID, Name: r.Name, Lang: r.Lang, Exts: r.Exts, Path: r.Path,
				Install: r.Install, OK: r.OK, Enabled: r.Enabled,
			})
		}
		a.set.lspRows = rows
		a.set.lspScanned, a.set.lspScanning = true, false
		return
	}
	a.set.lspScanning = true
	a.bg("settings.lsp", func() func() {
		defer diag.SpanSlow("settings", "检测语言服务器", 200*time.Millisecond)()
		rows := make([]lspSettingRow, 0, len(lsp.Specs))
		for i := range lsp.Specs {
			sp := &lsp.Specs[i]
			exts := strings.Join(sp.Exts, ", ")
			if len(exts) > 40 {
				exts = exts[:37] + "…"
			}
			path, ok := sp.Find()
			if !ok {
				path = ""
			} else {
				path = tildePath(path)
			}
			rows = append(rows, lspSettingRow{
				ID: sp.ID, Name: sp.Name, Lang: sp.Lang, Exts: exts,
				Path: path, Install: sp.Install, OK: ok,
				Enabled: a.prefs.Bool(a.lspServerPrefKey(sp.ID), true),
			})
		}
		return func() {
			a.set.lspRows = rows
			a.set.lspScanning = false
			a.set.lspScanned = true
		}
	})
}

func (a *App) lspInstallCmd(install string) {
	if a.proj == "" {
		a.toast = "请先打开一个项目"
		return
	}
	a.newTerm()
	a.ide.tpc().initCmd = install
	a.ide.bottomOpen, a.ide.bottomTab = true, "term"
	a.ide.termSel = ""
	a.outAppend(outLSP, "在终端安装:"+install)
	setLog.Info("语言服务器:在终端安装", "cmd", install)
}

func (a *App) lspBody(a2 *App, c *ui.Context) {
	t := a2.tk
	if !a2.set.lspScanned && !a2.set.lspScanning {
		a2.scanLspSettings()
	}
	if a2.set.lspScanning && len(a2.set.lspRows) == 0 {
		rowBox(a2, c, true, 40).Children(func() {
			ui.Text(c, "正在检测语言服务器…").FontSize(theme.FsCap).TextColor(t.Tx3)
		})
		return
	}
	globOn := a2.prefs.Bool("editor.lsp", true)
	for i, row := range a2.set.lspRows {
		row := row
		rowBox(a2, c, i == 0, 56).Children(func() {
			if row.OK {
				kit.Ic(c, "okc", kit.S12).TextColor(t.Ok)
			} else {
				kit.Ic(c, "alert", kit.S12).TextColor(t.Warn)
			}
			ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, row.Name).FontWeight(500)
				sub := row.Lang
				if row.Exts != "" {
					sub += " · " + row.Exts
				}
				ui.Text(c, sub).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
				if row.OK {
					ui.Text(c, row.Path).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
				} else if row.Install != "" {
					ui.Text(c, "未安装 · "+row.Install).FontSize(theme.FsCap).TextColor(t.Warn).SingleLine()
				} else {
					ui.Text(c, "未安装").FontSize(theme.FsCap).TextColor(t.Warn)
				}
			})
			ui.Text(c, "启用").FontSize(theme.FsCap).TextColor(t.Tx3)
			en := row.Enabled && globOn
			if kit.Tog(c, en, "lsp-en-"+row.ID) {
				want := !en
				if a2.persistPrefs() {
					if want {
						a2.prefs.Reset(a2.lspServerPrefKey(row.ID))
					} else {
						a2.prefs.Set(a2.lspServerPrefKey(row.ID), false)
					}
				}
				for j := range a2.set.lspRows {
					if a2.set.lspRows[j].ID == row.ID {
						a2.set.lspRows[j].Enabled = a2.prefs.Bool(a2.lspServerPrefKey(row.ID), true)
					}
				}
				setLog.Info("语言服务器启用", "id", row.ID, "on", want && globOn)
			}
			if !row.OK && row.Install != "" {
				if kit.Btn(c, "复制", kit.BtnOpt{Small: true, Icon: "copy"}).Clicked() {
					c.WriteClipboard(row.Install)
					a2.flash("已复制安装命令")
				}
				if kit.Btn(c, "在终端安装", kit.BtnOpt{Small: true, Icon: "terminal"}).Clicked() {
					a2.lspInstallCmd(row.Install)
				}
			}
		})
	}
	rowBox(a2, c, false, 36).Children(func() {
		if kit.Btn(c, "重新检测", kit.BtnOpt{Small: true, Icon: "refresh"}).Clicked() {
			a2.set.lspScanned = false
			a2.scanLspSettings()
		}
		hint := "在 PATH 与常见目录中查找"
		if !globOn {
			hint = "全局语言服务已关闭,开启后才会连接服务器"
		}
		ui.Text(c, hint).FontSize(theme.FsCap).TextColor(t.Tx3).Grow(1)
	})
}
