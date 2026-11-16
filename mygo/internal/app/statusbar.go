package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 状态栏的一个小项(.sbar .it:18 高、悬停 fill-1)。
func (a *App) sbItem(c *ui.Context, key string, children func()) ui.Element {
	t := a.tk
	b := ui.ButtonBase(c).Key("sb-" + key).Height(18).PaddingX(5).Gap(4).Radius(t.RXs()).AlignItems(ui.Center).Cursor(ui.CursorPointer).Shrink(0).Transition(kit.Fade)
	if b.Hovered() {
		b.Background(t.Fill1).TextColor(t.Tx1)
	}
	return b.Children(children)
}

// statusbar 是状态栏:两种形态共用,IDE 专属项在 IDE 形态才出现。
func (a *App) statusbar(c *ui.Context) {
	t := a.tk
	h := float32(theme.HStatus)
	if t.Scheme.ID == "b" {
		h = 24
	}
	r := ui.Row(c).Height(h).PaddingX(8).Gap(6).FontSize(theme.Px(11)).TextColor(t.Tx3).Shrink(0).AlignItems(ui.Center)
	if t.Scheme.ID != "b" {
		r.BorderWidth(1, 0, 0, 0).BorderColor(t.Bd2)
	}
	r.Children(func() {
		if a.branch != "" {
			a.sbItem(c, "branch", func() { kit.Ic(c, "branch", kit.S12); ui.Text(c, a.repoBranchLabel()) }).OnClick(func() { a.OpenQuick("branch", "") })
		}
		a.sbItem(c, "sync", func() {
			kit.Ic(c, "refresh", kit.S12)
			txt := "↑1" // 仓库状态读到之前沿用设计稿的示例
			if a.proj != "" {
				txt = "↑0"
			}
			if a.sc.loaded {
				txt = "↑" + itoa(a.sc.ahead)
				if a.sc.behind > 0 {
					txt += " ↓" + itoa(a.sc.behind)
				}
			}
			ui.Text(c, txt)
		}).OnClick(func() { a.OpenSCM() })
		a.sbItem(c, "diag", func() {
			ne, nw := 0, 1 // 演示数据沿用设计稿
			if a.proj != "" {
				ne, nw = a.lspCounts()
			}
			kit.Ic(c, "errc", kit.S12).TextColor(map[bool]ui.Color{true: t.Err, false: t.Tx3}[ne > 0])
			ui.Text(c, itoa(ne))
			kit.Ic(c, "alert", kit.S12).TextColor(map[bool]ui.Color{true: t.Warn, false: t.Tx3}[nw > 0])
			ui.Text(c, itoa(nw))
		}).OnClick(func() { a.ide.bottomOpen, a.ide.bottomTab = true, "problems" })
		if a.mode == ModeIDE {
			if tab := a.ide.cur(); tab != nil && !tab.diff && tab.st != nil {
				p := tab.st.CursorPos()
				// 点光标位置 = 转到行(与 VS Code 一致,键盘 ⌃G 同一入口)
				a.sbItem(c, "pos", func() { ui.Text(c, "行 "+itoa(p.Line+1)+",列 "+itoa(p.Col+1)).FontFeatures("tnum") }).OnClick(a.openGotoLine)
				indent := tab.st.TabSize
				if indent < 1 {
					indent = 4
				}
				if tab.st.Indent == editor.IndentTabs {
					a.sbItem(c, "indent", func() { ui.Text(c, "Tab") }).Tooltip("缩进方式(点击切换)").Menu(a.indentMenu)
				} else {
					a.sbItem(c, "indent", func() { ui.Text(c, "空格:"+itoa(indent)) }).Tooltip("缩进方式(点击切换)").Menu(a.indentMenu)
				}
				if n := a.prefs.Int(pEdFont, edFontDefault); a.prefs.Has(pEdFont) && n != edFontDefault {
					// 字号被改过:状态栏常驻提示,点一下恢复默认(⌘0)
					a.sbItem(c, "edfont", func() {
						ui.Text(c, "字号 "+itoa(n)+"px · 恢复")
					}).OnClick(a.resetEditorFont)
				}
				a.mcStatusChip(c, tab)
				if tab.diskRO {
					a.sbItem(c, "ro", func() { ui.Text(c, "只读") })
				}
				a.sbItem(c, "enc", func() { ui.Text(c, "UTF-8") }).Tooltip("文件编码(目前只支持 UTF-8)").Menu(a.encMenu)
				a.sbItem(c, "eol", func() { ui.Text(c, eolLabel(tab.eolNow())) }).Tooltip("行尾序列(点击切换)").Menu(a.eolMenu)
				lang := "纯文本"
				if l := tab.buf.Language(); l != nil {
					lang = l.Name
				}
				a.sbItem(c, "lang", func() { ui.Text(c, lang) })
				if a.proj != "" {
					a.lspStatus(c, tab)
				} else if srv := lspName(lang); srv != "" && a.demoExtra {
					a.sbItem(c, "lsp", func() { kit.Ic(c, "okc", kit.S12).TextColor(t.Ok); ui.Text(c, srv) })
				}
			}
		}
		ui.Spacer(c).Grow(1)
		a.connChip(c)
		mode := a.sbItem(c, "mode", func() {
			kit.Ic(c, modeIcon(a.st.Mode), kit.S12).TextColor(a.modeTint(a.st.Mode))
			ui.Text(c, a.st.Mode.Label())
		})
		mode.OnClick(func() { a.togglePop("sbcfg") })
		a.cfgPopup(c, "sbcfg", mode, true)
		mb := ui.ButtonBase(c).Key("sb-model").Height(18).PaddingX(5).Gap(4).Radius(t.RXs()).AlignItems(ui.Center).Cursor(ui.CursorPointer).Shrink(0).Transition(kit.Fade)
		if mb.Hovered() {
			mb.Background(t.Fill1).TextColor(t.Tx1)
		}
		if tip := a.pvModelChipTip(); tip != "" {
			mb.Tooltip(tip)
		}
		mb.Children(func() {
			if prov, mid := a.pvModelLabel(); prov != "" {
				_, pr := a.pvActive()
				pb := ""
				if pr != nil {
					pb = provBrand(pr.Name, pr.BaseURL)
				}
				kit.Logo(c, modelBrand(mid, pb), 13, t.Tx2)
			} else {
				kit.Logo(c, "claude", 13, t.Tx2)
			}
			ui.Text(c, a.modelName())
		})
		mb.OnClick(func() { a.togglePop("sbcfg") })
		ctx := a.sbItem(c, "ctx", func() {
			ui.Text(c, "上下文 "+itoa(int(a.st.Usage.ContextRatio()*100+.5))+"%").FontFeatures("tnum")
		})
		ctx.OnClick(func() { a.togglePop("ctx") })
		a.ctxPopup(c, ctx)
		// 已是最新时不占状态栏;只有检查中 / 有新版本 / 下载 / 失败等需要用户留意的状态才显示
		// (「已是最新」在设置 → 关于里查看)。
		if lbl := a.updLabel(); lbl != "" && a.updSt.Phase != updLatest {
			upd := a.sbItem(c, "upd", func() {
				kit.Ic(c, "download", kit.S12).TextColor(t.Ac)
				ui.Text(c, lbl).TextColor(t.Ac)
			})
			upd.OnClick(func() { a.togglePop("upd") })
			a.updPopup(c, upd)
		}
		if a.devBadge != "" && t.Scheme.ID != "x" {
			kit.Tag(c, a.devBadge, kit.KAc)
		}
	})
}

// lspName 返回某语言默认使用的语言服务器名(没有则为空)。
func lspName(lang string) string {
	switch lang {
	case "Rust":
		return "rust-analyzer"
	case "Go":
		return "gopls"
	case "Python":
		return "Pyright"
	case "TypeScript", "JavaScript":
		return "typescript-language-server"
	}
	return ""
}

// lspStatus 是状态栏里当前文件的语言服务状态:就绪 / 启动中(带索引进度)/ 没装 / 失败;点击打开问题面板(安装与重试在那里)。
func (a *App) lspStatus(c *ui.Context, tab *ideTab) {
	t := a.tk
	s := a.lspFor(tab.path)
	if s == nil {
		return
	}
	it := a.sbItem(c, "lsp", func() {
		switch s.state {
		case lspReady:
			kit.Ic(c, "okc", kit.S12).TextColor(t.Ok)
			ui.Text(c, s.spec.Name)
			if s.work != "" {
				ui.Text(c, s.work).TextColor(t.Tx4).SingleLine().MaxWidth(160)
			}
		case lspStarting:
			kit.Spin(c, t.Tx3)
			line := s.spec.Name + " 启动中…"
			if w := strings.TrimSpace(s.work); w != "" {
				line = s.spec.Name + " " + w
			}
			ui.Text(c, line).TextColor(t.Tx3).SingleLine().MaxWidth(200)
		case lspMissing:
			kit.Ic(c, "alert", kit.S12).TextColor(t.Warn)
			ui.Text(c, "未安装 "+s.spec.Name).TextColor(t.Warn)
		default:
			kit.Ic(c, "errc", kit.S12).TextColor(t.Err)
			ui.Text(c, s.spec.Name+" 未运行").TextColor(t.Err)
		}
	})
	it.OnClick(func() { a.ide.bottomOpen, a.ide.bottomTab = true, "problems" })
}
