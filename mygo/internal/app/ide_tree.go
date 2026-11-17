package app

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// filesView 是 IDE 左栏的「文件」:区块标题 + 文件树 + 折叠的「大纲 / 时间线」。
func (a *App) filesView(c *ui.Context) {
	if a.proj != "" {
		a.scmTick() // 真实项目:树上的 Git 状态色靠它定时刷新
	}
	t := a.tk
	d := a.ide
	d.treeIdx, d.treeHit = 0, -1
	// 外层竖排:文件树自己滚动,「大纲 / 时间线」像 VS Code 的分区那样钉在底部。
	outer := ui.Column(c).Grow(1).MinHeight(0)
	var secBox ui.Element
	haveSec := false
	outer.Children(func() {
		files := kit.VScroll(c).Key("files").TrackScroll(&d.treeSt).Grow(1).MinHeight(0)
		files.OnShortcut(ui.Cmd, ui.KeyF, a.toggleTreeFilter)
		files.OnShortcut(0, ui.KeyUp, func() { a.treeFilterStep(-1) })
		files.OnShortcut(0, ui.KeyDown, func() { a.treeFilterStep(1) })
		files.OnShortcut(0, ui.KeyF2, func() {
			if a.ide.sel == "" {
				return
			}
			a.filesFocus = true
			a.beginRenameFile(a.ide.sel)
		})
		files.Children(func() {
			ui.Column(c).Children(func() {
				a.openedEditorsSection(c)
				hdr := ui.Row(c).Key("tree-hdr").Height(theme.Px(26)).PaddingX(8).Gap(5).AlignItems(ui.Center).TextColor(t.Tx3).FontSize(theme.Px(11.5)).FontWeight(600)
				hdrHot := hdr.Hovered() || a.popup != ""
				hdr.Children(func() {
					title := strings.ToUpper(d.fs.Name())
					if a.multiRoot() {
						title = strings.ToUpper(a.ws.Name) + "(工作区)"
					}
					// 标题 + 箭头整块可点,点一下折叠 / 展开整棵文件树(与「大纲 / 时间线」区块一致)
					head := ui.ButtonBase(c).Key("tree-root-head").Grow(1).Height(theme.Px(26)).Justify(ui.Start).Gap(5).AlignItems(ui.Center).
						TextColor(t.Tx3).FontSize(theme.Px(11.5)).FontWeight(500).Cursor(ui.CursorDefault)
					if hdrHot {
						// 工具按钮叠在行尾且没有底色(导航栏底色随主题 / 方案不同,垫一块纯色会露出色差):
						// 显示时给标题让出同样宽的位置,让标题自己截断,而不是被按钮压住
						head.PaddingEnd(6*theme.Px(20) + 5 + 10 + 4)
					}
					if head.Clicked() {
						d.rootShut = !d.rootShut
						diag.L("ide").Info("折叠文件树根", "折叠", d.rootShut)
					}
					head.Children(func() {
						icon := "chevD"
						if d.rootShut {
							icon = "chevR"
						}
						kit.Ic(c, icon, kit.S12)
						ui.Text(c, title).Grow(1).SingleLine()
					})
					// 图标各有各的含义:添加文件夹 / 新建文件夹 = 带加号的文件夹,新建文件 = 带加号的文件,
					// 全部收起 = 两枚相对的箭头夹一条线;多根时「新建文件夹」只留在 ··· 菜单(顶层要先选文件夹)
					_, ro := a.mutable()
					// 工具按钮默认隐藏,鼠标移到标题行上才淡入(叠在行尾,标题文字平时完整显示);按钮始终存在,
					// 所以从 ··· 打开菜单后鼠标移走也不会丢掉菜单的锚点
					tools := ui.Row(c).Key("tree-tools").Absolute().Right(0).Top(0).Bottom(0).PaddingStart(10).PaddingEnd(4).Gap(1).AlignItems(ui.Center).
						Transition(kit.Fade)
					if hdrHot {
						tools.Opacity(1)
					} else {
						tools.Opacity(0)
					}
					tools.Children(func() {
						if a.multiRoot() {
							kit.Ib(c, "folderPlus", kit.IbOpt{Small: true, Tip: "添加文件夹到工作区"}).OnClick(func() {
								a.pickFolders(true, func(ps []string) { a.wsAddFolders(a.ws.ID, ps) })
							})
						}
						newFile := kit.Ib(c, "filePlus", kit.IbOpt{Small: true, Tip: map[bool]string{true: "新建文件", false: ro}[ro == ""]})
						if ro == "" {
							newFile.OnClick(func() { a.beginNew(a.newTargetDir(), false) })
						} else {
							newFile.Opacity(0.4).Cursor(ui.CursorDefault)
						}
						if !a.multiRoot() {
							newDir := kit.Ib(c, "folderPlus", kit.IbOpt{Small: true, Tip: map[bool]string{true: "新建文件夹", false: ro}[ro == ""]})
							if ro == "" {
								newDir.OnClick(func() { a.beginNew(a.newTargetDir(), true) })
							} else {
								newDir.Opacity(0.4).Cursor(ui.CursorDefault)
							}
						}
						flt := kit.Ib(c, "filter", kit.IbOpt{Small: true, Tip: map[bool]string{true: "关闭过滤", false: "过滤文件(⌘F)"}[d.fltOpen]})
						flt.OnClick(a.toggleTreeFilter)
						shut := kit.Ib(c, "collapseAll", kit.IbOpt{Small: true, Tip: map[bool]string{true: "全部收起", false: "没有展开的文件夹"}[a.treeOpenCount() > 0]})
						if a.treeOpenCount() > 0 {
							shut.OnClick(a.collapseTree)
						} else {
							shut.Opacity(0.4).Cursor(ui.CursorDefault)
						}
						kit.Ib(c, "refresh", kit.IbOpt{Small: true, Tip: "刷新"}).OnClick(func() { d.listCache, d.filesCache = nil, nil })
						kit.Ib(c, "more", kit.IbOpt{Small: true, Tip: "更多操作"}).Menu(a.treeHeadMenu)
					})
				})
				a.treeFilterBar(c)
				if !d.rootShut {
					if d.fltActive() {
						a.treeFiltered(c, "", 0)
					} else {
						lines := a.treeLineCache()
						d.treeHit = -1
						if d.revealTo != "" {
							for i, ln := range lines {
								if treeLinePath(ln) == d.revealTo {
									d.treeHit = i
									break
								}
							}
						}
						ui.List(c, &d.treeList, len(lines), func(i int) {
							if i < len(lines) {
								a.treeLineAt(c, lines[i], i)
							}
						}).Grow(1).MinHeight(0)
					}
				}
			})
		})
		outerH := outer.Bounds().H
		secBox, haveSec = ui.Column(c).Shrink(0), true
		secBox.Children(func() {
			ui.Box(c).Height(1).Background(t.Bd1)
			for _, name := range []string{"大纲", "时间线"} {
				open := d.sections[name]
				lyKey := lySideOutline
				defH := sideOutlineDef
				if name == "时间线" {
					lyKey, defH = lySideTimeline, sideTimelineDef
				}
				bodyH := a.sideSecHeight(lyKey, defH, outerH)
				ui.Column(c).Key("sec-wrap-" + name).Shrink(0).Children(func() {
					if open {
						a.resizeHandle(c, lyKey, "t", bodyH, defH, sideSecMin, a.sideSecMax(outerH), func(v float32) {
							switch lyKey {
							case lySideOutline:
								a.ly.sideOutline = v
							default:
								a.ly.sideTimeline = v
							}
						})
					}
					hdr := ui.Row(c).Key("sec-h-" + name).Height(theme.Px(26)).PaddingX(8).Gap(5).AlignItems(ui.Center).TextColor(t.Tx3).
						FontSize(theme.Px(11.5)).FontWeight(500).Shrink(0)
					hdr.Children(func() {
						tog := ui.ButtonBase(c).Key("sec-" + name).Grow(1).Height(theme.Px(26)).Justify(ui.Start).Gap(5).AlignItems(ui.Center).
							TextColor(t.Tx3).FontSize(theme.Px(11.5)).FontWeight(500).Cursor(ui.CursorDefault)
						if tog.Clicked() {
							d.sections[name] = !open
							diag.L("ide").Info("折叠侧栏区块", "区块", name, "展开", !open)
						}
						tog.Children(func() {
							icon := "chevR"
							if open {
								icon = "chevD"
							}
							kit.Ic(c, icon, kit.S12)
							ui.Text(c, name)
						})
						if name == "时间线" && open {
							a.timelineHeaderTools(c)
						}
					})
					if open {
						body := ui.Column(c).Key("sec-body-" + name).Height(bodyH).Shrink(0)
						body.Children(func() {
							scroll := kit.VScroll(c).Key("sec-scroll-" + name).Grow(1).MinHeight(0)
							scroll.OnShortcut(0, ui.KeyUp, func() {
								if name == "时间线" {
									a.timelineStep(-1)
								}
							})
							scroll.OnShortcut(0, ui.KeyDown, func() {
								if name == "时间线" {
									a.timelineStep(1)
								}
							})
							scroll.Children(func() {
								ui.Column(c).Children(func() {
									if name == "大纲" {
										a.outlineRows(c)
									} else {
										a.timelineRows(c)
									}
								})
							})
						})
					}
				})
			}
		})
	})
	// 文件树视口高 ≈ 外层高 − 钉在底部的区块高(都取上一帧);然后把待定位的行滚进可见范围
	if h := outer.Bounds().H; h > 0 && haveSec {
		d.treeH = h - secBox.Bounds().H
	}
	a.treeRevealScroll()
}

// listing 带缓存地列目录:界面每帧都在画树,不能每帧读磁盘。刷新按钮 / 文件变化清缓存。
func (d *ide) listing(dir string) []workspace.Entry {
	if d.listCache == nil {
		d.listCache = map[string][]workspace.Entry{}
	}
	if es, ok := d.listCache[dir]; ok {
		return es
	}
	es, err := d.fs.List(dir)
	if err != nil {
		diag.L("ide").Warn("列目录失败", "dir", dir, "err", err)
	}
	d.listCache[dir] = es
	return es
}

func (a *App) treeDir(c *ui.Context, dir string, depth int) {
	d := a.ide
	if o := a.fo; (o.kind == "new" || o.kind == "newdir") && o.dir == dir && !(dir == "" && a.multiRoot()) {
		a.fileOpInput(c, depth, o.kind == "newdir", "file") // 新建:输入行插在这个文件夹子项的最前
	}
	for _, e := range a.ide.listing(dir) {
		if e.Dir && a.fo.kind != "rename" && a.fo.path != e.Path {
			if tail, label, compact := a.ide.compactFolderTail(e.Path); compact {
				a.treeCompactRow(c, tail, label, depth, d.treeIdx)
				d.treeIdx++
				continue
			}
		}
		if a.fo.kind == "rename" && a.fo.path == e.Path { // 重命名:这一行换成输入框
			a.fileOpInput(c, depth, e.Dir, fileIcon(e.Name))
			if e.Dir && a.ide.open[e.Path] {
				a.treeDir(c, e.Path, depth+1)
			}
			continue
		}
		a.treeRow(c, e, depth, 0)
		if e.Dir && a.ide.open[e.Path] {
			a.treeDir(c, e.Path, depth+1)
		}
	}
}

func (a *App) treeRow(c *ui.Context, e workspace.Entry, depth, row int) {
	t := a.tk
	d := a.ide
	on := d.sel == e.Path // 文件夹点过之后也高亮,否则只看得到文件被选中
	rowEl := kit.RowEl(c, on).Key("tr-" + e.Path)
	var r ui.Element
	if d.fltActive() {
		if e.Path == d.revealTo {
			d.treeHit = d.treeIdx
		}
		d.treeIdx++
		r = a.navDrop(rowEl)
	} else {
		r = a.treeNavDrop(rowEl, row)
	}
	r = r.PaddingStart(6 + float32(depth)*14).Gap(6)
	dragRef(r, e.Path, e.Dir) // 拖到对话栏 = 引用
	if e.Dir {
		if ref, ok := ui.Drop[fileRef](r); ok && !ref.Dir {
			a.moveFile(ref.Path, e.Path)
		}
	}
	r.ContextMenu(func(m *ui.Menu) { a.treeMenu(m, e.Path, e.Dir) })
	g := d.fileGit(e.Path)
	var dagg dirGitAgg
	if e.Dir {
		if a.ide.dirGit != nil {
			dagg = a.ide.dirGit[e.Path]
		}
		if dagg.hasChange() {
			r.TextColor(treeGitColor(t, dagg.worst))
		} else if treeDirMuted(e.Name, e.Path, dagg) {
			r.TextColor(t.Tx4)
		} else {
			r.TextColor(t.Tx1)
		}
	} else {
		switch g {
		case "M", "R":
			r.TextColor(t.Warn)
		case "A", "U":
			r.TextColor(t.Ok)
		case "D", "C":
			r.TextColor(t.Err)
		}
	}
	if r.Clicked() {
		a.filesFocus = true
		d.sel = e.Path
		if e.Dir {
			d.open[e.Path] = !d.open[e.Path]
		} else {
			d.openFile(e.Path, false)
		}
	}
	if r.DoubleClicked() && !e.Dir {
		d.openFile(e.Path, true)
	}
	r.Children(func() {
		if on {
			kit.SelBar(c)
		}
		if e.Dir {
			icon := "chevR"
			if d.open[e.Path] || d.fltActive() { // 过滤时目录一律展开(命中项藏在里面)
				icon = "chevD"
			}
			kit.Ic(c, icon, kit.S12).TextColor(t.Tx3)
			kit.Ic(c, treeFolderIcon(e.Name), kit.S14).TextColor(map[bool]ui.Color{true: t.Tx4, false: t.Tx3}[treeDirMuted(e.Name, e.Path, dagg)])
		} else {
			ui.Box(c).Size(theme.Px(12), theme.Px(12)).Shrink(0)
			a.fileIc(c, e.Name)
		}
		plain := !d.fltActive()
		if d.fltActive() {
			a.treeHighlightName(c, e.Name, strings.TrimSpace(d.flt))
		} else {
			nm := ui.Text(c, e.Name).Grow(1).SingleLine()
			if depth == 0 && e.Dir && a.multiRoot() { // 多根:顶层是各个文件夹,加粗并标注改动数 / 非仓库
				nm.FontWeight(500)
			}
			_ = nm
		}
		if plain && depth == 0 && e.Dir && a.multiRoot() { // 徽章仍要画
			a.rootBadges(c, e.Name)
		} else if e.Dir && a.multiRoot() {
			a.nestedBadges(c, e.Path) // 嵌套仓库 / 子模块的根目录:类别标签 + 改动数
		}
		if g != "" && !e.Dir {
			ui.Text(c, g).FontSize(theme.Px(10)).FontWeight(600).Shrink(0)
		}
		if e.Dir && dagg.hasChange() {
			dot := ui.Box(c).Size(6, 6).Radius(3).Shrink(0).Background(treeGitColor(t, dagg.worst))
			if tip := dagg.tooltip(); tip != "" {
				dot.Tooltip(tip)
			}
		}
	})
}

// fileIcon 按扩展名挑图标(设计稿:源码 code、配置 gear、其余 fileText)。
func fileIcon(name string) string {
	switch {
	case strings.HasSuffix(name, ".rs"), strings.HasSuffix(name, ".go"), strings.HasSuffix(name, ".ts"),
		strings.HasSuffix(name, ".js"), strings.HasSuffix(name, ".py"):
		return "code"
	case strings.HasSuffix(name, ".toml"), strings.HasSuffix(name, ".json"), strings.HasSuffix(name, ".yaml"), strings.HasSuffix(name, ".yml"):
		return "gear"
	case strings.HasSuffix(name, ".png"), strings.HasSuffix(name, ".jpg"), strings.HasSuffix(name, ".jpeg"), strings.HasSuffix(name, ".webp"), strings.HasSuffix(name, ".gif"), strings.HasSuffix(name, ".svg"):
		return "image"
	}
	return "fileText"
}

// allFiles 递归列出项目里的全部文件(相对路径,已排序),命令面板的文件搜索用;
// 跳过隐藏项与常见产物目录,总数封顶 5000,结果缓存到文件树刷新。
func (d *ide) allFiles() []string {
	if d.filesCache != nil {
		return d.filesCache
	}
	defer diag.SpanSlow("ide", "枚举项目文件", 50*time.Millisecond)()
	out := []string{}
	var walk func(dir string)
	walk = func(dir string) {
		if len(out) >= 5000 {
			return
		}
		es, err := d.fs.List(dir)
		if err != nil {
			return
		}
		for _, e := range es {
			if strings.HasPrefix(e.Name, ".") || e.Name == "target" || e.Name == "node_modules" {
				continue
			}
			if e.Dir {
				walk(e.Path)
			} else {
				out = append(out, e.Path)
			}
		}
	}
	walk("")
	d.filesCache = out
	return out
}

// outlineRows 是左栏「大纲」:当前文件的符号树(来自文本符号索引,Markdown 取标题),点击跳转并高亮光标所在项。
func (a *App) outlineRows(c *ui.Context) {
	t := a.tk
	tb := a.ide.cur()
	if tb == nil || tb.virt != "" {
		ui.Text(c, "打开一个文件查看大纲").FontSize(theme.FsCap).TextColor(t.Tx4).Margin(2, 8, 6, 28)
		return
	}
	if tb.diff || tb.buf == nil || tb.st == nil {
		syms := a.symbolsForFile(tb.path)
		if len(syms) == 0 {
			ui.Text(c, "这个文件没有可识别的符号").FontSize(theme.FsCap).TextColor(t.Tx4).Margin(2, 8, 6, 28)
			return
		}
		for i, s := range syms {
			depth := 0
			for k := s.Parent; k >= 0; k = syms[k].Parent {
				depth++
			}
			i, s := i, s
			r := kit.RowEl(c, false).Key("ol-" + itoa(i)).PaddingStart(22 + float32(depth)*12).Gap(5)
			if r.Clicked() && tb.dv != nil {
				a.jumpTo(tb.path, s.Line, s.Col, s.Col)
			}
			r.Children(func() {
				icon := "cube"
				if s.Kind == "h" {
					icon = "fileText"
				}
				kit.Ic(c, icon, kit.S12).TextColor(t.AcTx)
				ui.Text(c, s.Name).Grow(1).SingleLine()
				ui.Text(c, itoa(s.Line+1)).FontSize(theme.FsMicro).TextColor(t.Tx4)
			})
		}
		return
	}
	syms := a.symbolsForFile(tb.path)
	if len(syms) == 0 {
		ui.Text(c, "这个文件没有可识别的符号").FontSize(theme.FsCap).TextColor(t.Tx4).Margin(2, 8, 6, 28)
		return
	}
	cur := -1
	if ch := enclosing(syms, tb.st.CursorPos().Line); len(ch) > 0 {
		cur = ch[len(ch)-1]
	}
	for i, s := range syms {
		depth := 0
		for k := s.Parent; k >= 0; k = syms[k].Parent {
			depth++
		}
		i, s := i, s
		r := kit.RowEl(c, i == cur).Key("ol-" + itoa(i)).PaddingStart(22 + float32(depth)*12).Gap(5)
		if r.Clicked() {
			a.jumpTo(tb.path, s.Line, s.Col, s.Col)
		}
		r.Children(func() {
			icon := "cube"
			if s.Kind == "h" {
				icon = "fileText"
			}
			kit.Ic(c, icon, kit.S12).TextColor(t.AcTx)
			ui.Text(c, s.Name).Grow(1).SingleLine()
			ui.Text(c, itoa(s.Line+1)).FontSize(theme.FsMicro).TextColor(t.Tx4)
		})
	}
}

// fileTint 是文件图标的颜色:按类型取主题令牌(换主题自动跟着变)。
// 设置 → 外观 →「彩色文件图标」关闭后一律用次要文字色。
func (a *App) fileTint(name string, fallback ui.Color) ui.Color {
	if !a.prefs.Bool("appearance.fileIcons", true) {
		return fallback
	}
	t := a.tk
	n := strings.ToLower(name)
	switch {
	case strings.HasSuffix(n, ".rs"):
		return t.Warn
	case strings.HasSuffix(n, ".go"):
		return t.Info
	case strings.HasSuffix(n, ".ts"), strings.HasSuffix(n, ".tsx"), strings.HasSuffix(n, ".swift"):
		return t.Ac
	case strings.HasSuffix(n, ".js"), strings.HasSuffix(n, ".jsx"), strings.HasSuffix(n, ".lua"):
		return t.Warn
	case strings.HasSuffix(n, ".py"):
		return t.Ok
	case strings.HasSuffix(n, ".png"), strings.HasSuffix(n, ".jpg"), strings.HasSuffix(n, ".jpeg"), strings.HasSuffix(n, ".webp"), strings.HasSuffix(n, ".gif"), strings.HasSuffix(n, ".svg"):
		return t.Vio
	case strings.HasSuffix(n, ".md"), strings.HasSuffix(n, ".txt"):
		return t.Tx3
	case strings.HasSuffix(n, ".toml"), strings.HasSuffix(n, ".json"), strings.HasSuffix(n, ".yaml"), strings.HasSuffix(n, ".yml"):
		return t.Tx2
	case strings.HasSuffix(n, ".sh"), strings.HasSuffix(n, ".zsh"):
		return t.Ok
	case strings.HasSuffix(n, ".c"), strings.HasSuffix(n, ".h"), strings.HasSuffix(n, ".cpp"), strings.HasSuffix(n, ".hpp"):
		return t.Info
	case n == "cargo.lock", strings.HasSuffix(n, ".lock"), strings.HasPrefix(n, "."):
		return t.Tx4
	}
	return fallback
}

// fileIc 画文件图标:认识的语言用徽标(file_icon.go),其余是带类型色的线条图标。
func (a *App) fileIc(c *ui.Context, name string) ui.Element {
	if a.prefs.Bool("appearance.fileIcons", true) {
		if k, ok := fileKindOf(name); ok {
			return a.fileBadge(c, k)
		}
	}
	return kit.Ic(c, fileIcon(name), kit.S12).TextColor(a.fileTint(name, a.tk.Tx3))
}

// treeOpenCount 文件树里当前展开的文件夹数。
func (a *App) treeOpenCount() int {
	n := 0
	for _, on := range a.ide.open {
		if on {
			n++
		}
	}
	return n
}

// collapseTree 一键收起整棵文件树(标题行按钮与 ··· 菜单共用);选中项若落在被收起的文件夹里,改选它所在的最外层文件夹,避免选中一个看不见的行。
func (a *App) collapseTree() {
	d := a.ide
	n := a.treeOpenCount()
	for k := range d.open {
		delete(d.open, k)
	}
	if d.sel != "" {
		if top, _, ok := strings.Cut(d.sel, "/"); ok {
			d.sel = top
		}
	}
	diag.L("ide").Info("全部收起文件树", "收起", n)
}
