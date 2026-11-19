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

	Managed bool   // 支持下载安装(清单里固定了 npm 包)
	MgrPath bool   // 找到的是应用下载安装的那份(~/.minicode/lsp/…),可以卸载
	Stale   bool   // 托管目录里的版本与清单固定的不一致:有新版本可更新
	Node    string // 托管的服务器用的是哪个 Node,如「本机 v24.11.1」「托管 v24.11.1」
}

func (a *App) lspServerPrefKey(id string) string { return "lsp.server." + id }

func (a *App) scanLspSettings() {
	if a.demoWs() {
		var rows []lspSettingRow
		for _, r := range demo.LSPDemoRows() {
			row := lspSettingRow{
				ID: r.ID, Name: r.Name, Lang: r.Lang, Exts: r.Exts, Path: r.Path,
				Install: r.Install, OK: r.OK, Enabled: r.Enabled,
			}
			if sp := lspSpecByID(r.ID); sp != nil {
				row.Managed = sp.Managed()
				row.MgrPath = r.OK && strings.Contains(r.Path, ".minicode/lsp/")
				if row.MgrPath {
					row.Node = "本机 " + lsp.ManagedNodeVersion
					row.Stale = r.Status == "可更新"
				}
			}
			rows = append(rows, row)
		}
		a.set.lspRows = rows
		a.set.lspScanned, a.set.lspScanning = true, false
		a.lspSeedDemoJobs()
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
			mgr := ok && lsp.IsManagedPath(path)
			node := ""
			if mgr { // 托管的服务器靠 Node 跑:写清用的是哪一个
				if n, found := lsp.ResolveNode(sp.MinNode, false); !found {
					node = "未找到 Node"
				} else if n.Managed {
					node = "托管 " + n.Version
				} else {
					node = "本机 " + n.Version
				}
			}
			if !ok {
				path = ""
			} else {
				path = tildePath(path)
			}
			rows = append(rows, lspSettingRow{
				ID: sp.ID, Name: sp.Name, Lang: sp.Lang, Exts: exts,
				Path: path, Install: sp.Install, OK: ok,
				Enabled: a.prefs.Bool(a.lspServerPrefKey(sp.ID), true),
				Managed: sp.Managed(), MgrPath: mgr, Stale: mgr && sp.ManagedStale(), Node: node,
			})
		}
		return func() {
			a.set.lspRows = rows
			a.set.lspScanning = false
			a.set.lspScanned = true
		}
	})
}

// lspSeedDemoJobs 演示数据:一个失败(可重试)、一个下载中(可取消),设计稿里的各态都能直接看到。
func (a *App) lspSeedDemoJobs() {
	if a.lspJobs != nil {
		return
	}
	a.lspJobs = map[string]*lspJob{}
	if sp := lspSpecByID("ts"); sp != nil {
		a.lspJobs["ts"] = &lspJob{sp: sp, err: "下载 Node 失败:连接超时(nodejs.org)· 可以重试,或在下方换成国内镜像"}
		a.lspJobOrder = append(a.lspJobOrder, "ts")
	}
	if sp := lspSpecByID("php"); sp != nil {
		a.lspJobs["php"] = &lspJob{sp: sp, running: true, stage: "下载 Node " + lsp.ManagedNodeVersion, done: 21 << 20, total: 49 << 20}
		a.lspJobOrder = append(a.lspJobOrder, "php")
	}
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
		sp := lspSpecByID(row.ID)
		var job *lspJob
		if sp != nil {
			job = a2.lspJobOf(sp)
		}
		busy := job != nil && job.err == "" // 下载 / 安装中或排队中
		failed := job != nil && job.err != ""
		rowBox(a2, c, i == 0, 56).Children(func() {
			switch {
			case busy:
				kit.Spin(c, t.Ac)
			case failed:
				kit.Ic(c, "errc", kit.S12).TextColor(t.Err)
			case row.OK:
				kit.Ic(c, "okc", kit.S12).TextColor(t.Ok)
			default:
				kit.Ic(c, "alert", kit.S12).TextColor(t.Warn)
			}
			ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, row.Name).FontWeight(500)
					if row.Stale && !busy {
						ui.Text(c, "有新版本").FontSize(theme.FsMicro).TextColor(t.Ac)
					}
				})
				sub := row.Lang
				if row.Exts != "" {
					sub += " · " + row.Exts
				}
				ui.Text(c, sub).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
				switch {
				case busy:
					ui.Text(c, lspJobText(job)).FontSize(theme.FsCap).TextColor(t.Tx2).SingleLine()
					if p := job.pct(); p >= 0 && job.running {
						ui.Row(c).Height(3).Radius(2).Background(t.Fill2).Children(func() {
							ui.Box(c).Grow(float32(p)).Basis(0).Background(t.Ac).Radius(2)
							ui.Box(c).Grow(float32(100 - p)).Basis(0)
						})
					}
				case failed:
					ui.Text(c, job.err).FontSize(theme.FsCap).TextColor(t.Err)
				case row.OK:
					path := row.Path
					if row.MgrPath && row.Node != "" {
						path += " · Node:" + row.Node
					}
					ui.Text(c, path).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
				case row.Install != "":
					ui.Text(c, "未安装 · "+row.Install).FontSize(theme.FsCap).TextColor(t.Warn).SingleLine()
				default:
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
			switch {
			case busy:
				if kit.Btn(c, "取消", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).Clicked() {
					a2.lspJobCancel(sp)
				}
			case failed:
				if kit.Btn(c, "重试", kit.BtnOpt{Small: true, Variant: kit.BtnSoft, Icon: "refresh"}).Clicked() {
					a2.lspInstallManaged(sp)
				}
				if kit.Btn(c, "查看日志", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).Clicked() {
					a2.lspShowLog()
				}
				if kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "关闭这条错误"}).Clicked() {
					a2.lspJobDismiss(sp)
				}
			case row.OK && row.MgrPath:
				if row.Stale {
					if kit.Btn(c, "更新", kit.BtnOpt{Small: true, Variant: kit.BtnSoft, Icon: "download"}).Clicked() {
						a2.lspInstallManaged(sp)
					}
				}
				if kit.Btn(c, "卸载", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "trash"}).Tooltip("删除应用下载安装的这份(Node 运行时保留)").Clicked() {
					a2.lspUninstallManaged(sp)
				}
			case !row.OK && row.Install != "":
				if row.Managed {
					if kit.Btn(c, "下载安装", kit.BtnOpt{Small: true, Variant: kit.BtnSoft, Icon: "download"}).
						Tooltip("用 " + a2.lspMirror().Label() + " 下载固定版本的语言包;本机没有合适的 Node 时会先下载一份(点了才联网)").Clicked() {
						a2.lspInstallManaged(sp)
					}
				}
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
	rowBox(a2, c, false, 44).Children(func() {
		ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
			ui.Text(c, "下载源").FontWeight(500)
			ui.Text(c, "Node 运行时与语言包从哪里下载,切换只影响之后的下载").FontSize(theme.FsCap).TextColor(t.Tx3)
		})
		on := 0
		if a2.lspMirror() == lsp.MirrorCN {
			on = 1
		}
		if i := kit.Seg(c, []kit.SegItem{{Text: lsp.MirrorOfficial.Label()}, {Text: lsp.MirrorCN.Label()}}, on, kit.SegOpt{Key: "lsp-mirror"}); i >= 0 && i != on {
			m := lsp.MirrorOfficial
			if i == 1 {
				m = lsp.MirrorCN
			}
			if a2.persistPrefs() {
				a2.setLspMirror(m)
			}
		}
	})
	rowBox(a2, c, false, 36).Children(func() {
		if kit.Btn(c, "重新检测", kit.BtnOpt{Small: true, Icon: "refresh"}).Clicked() {
			a2.set.lspScanned = false
			a2.scanLspSettings()
		}
		hint := "先在 PATH 与常见目录中查找,没有再用下载安装的"
		if !globOn {
			hint = "全局语言服务已关闭,开启后才会连接服务器"
		}
		ui.Text(c, hint).FontSize(theme.FsCap).TextColor(t.Tx3).Grow(1)
	})
}
