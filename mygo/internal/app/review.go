package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diff"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 「审查 · 本轮改动」标签:Claude 这一轮改了什么——逐文件、逐块接受 / 撤销 / 评论。
// 这是 Claude 改动的唯一的家;会话流末尾的改动卡与 SCM 顶部的「审查本轮」都只是跳到这里。
// 接受 = 保留(不写 Git);撤销 = 把这一块还原成改动前(直接改工作区文件)。

// chFile 是「改动」里的一个文件。
type chFile struct {
	path     string
	status   string // M / A
	add, del int
	blocks   []*core.DiffBlock // 这一轮对该文件的每一处差异(每处 = 一个块)
}

// reviewState 是审查标签的界面状态。
type reviewState struct {
	open       map[string]bool   // 文件是否展开(没有记录 = 只有第一个文件默认展开)
	viewed     map[string]bool   // 已查看的文件
	hunk       map[string]string // 块的处理结果:ok 已接受 / rev 已撤销
	commenting string            // 正在写评论的块
	comments   map[string]string // 块 → 评论(只留在这里,「发送给 Claude」才会作为下一条消息)
	draft      string
	side       bool            // 左右对比(默认行内)
	grpClosed  map[string]bool // 多仓库分组是否折叠(键 g<序号>)
}

func (r *reviewState) init() {
	if r.open == nil {
		r.open, r.viewed, r.hunk, r.comments = map[string]bool{}, map[string]bool{}, map[string]string{}, map[string]string{}
		r.grpClosed = map[string]bool{}
	}
}

// reviewStat 一个块的增删行数。
func reviewStat(d *core.DiffBlock) (int, int) {
	return diff.Stat(diff.Lines(d.Old, d.New, d.Line, d.Line))
}

func hunkKey(path string, i int) string { return path + "#" + itoa(i) }

// changedFiles 汇总会话里 Claude 改过的文件(按首次出现顺序)。
func (a *App) changedFiles() []chFile {
	if a.reviewDemoEmpty {
		return nil
	}
	idx := map[string]int{}
	var out []chFile
	var visit func(items []core.Item)
	visit = func(items []core.Item) {
		for _, it := range items {
			switch v := it.(type) {
			case *core.ToolCall:
				for _, ct := range v.Content {
					d, ok := ct.(*core.DiffBlock)
					if !ok {
						continue
					}
					ad, de := diff.Stat(diff.Lines(d.Old, d.New, d.Line, d.Line))
					i, seen := idx[d.Path]
					if !seen {
						st := "M"
						if v.Kind == core.KindWrite {
							st = "A"
						}
						idx[d.Path] = len(out)
						out = append(out, chFile{path: d.Path, status: st})
						i = len(out) - 1
					}
					out[i].add += ad
					out[i].del += de
					out[i].blocks = append(out[i].blocks, d)
				}
			case *core.SubagentCard:
				visit(v.Children)
			}
		}
	}
	visit(a.st.Items)
	// 演示:工作区里还有一个没被本会话碰过的未提交文件,与设计稿的「3 个文件」一致
	if a.demoExtra {
		src, _ := a.ide.fs.Read(a.wsRel("tests/case_test.rs"))
		blk := &core.DiffBlock{Path: "tests/case_test.rs", New: src, Line: 1}
		ad, _ := diff.Stat(diff.Lines("", src, 1, 1))
		out = append(out, chFile{path: blk.Path, status: "A", add: ad, blocks: []*core.DiffBlock{blk}})
	}
	out = append(out, a.demoRepoFiles()...) // 演示工作区:另外两个文件夹也各有一轮改动
	return out
}

// fileDecided 文件的所有块是否都已处理。
func (a *App) fileDecided(f chFile) bool {
	for i := range f.blocks {
		if a.rv.hunk[hunkKey(f.path, i)] == "" {
			return false
		}
	}
	return len(f.blocks) > 0
}

// pendingReviewFiles 还需要处理的文件数(标签角标)。
func (a *App) pendingReviewFiles() int {
	n := 0
	for _, f := range a.changedFiles() {
		if !a.fileDecided(f) && !a.rv.viewed[f.path] {
			n++
		}
	}
	return n
}

// OpenReview 打开审查标签(会话流改动卡「审查」、SCM「审查本轮」的入口)。
func (a *App) OpenReview() { a.OpenWorkspaceTab(vReview, "") }

// hunkRange 块在新文件里的行范围。
func hunkRange(d *core.DiffBlock) (int, int) {
	ls := diff.Lines(d.Old, d.New, d.Line, d.Line)
	first, last := 0, 0
	for _, l := range ls {
		if l.NewNo == 0 {
			continue
		}
		if first == 0 {
			first = l.NewNo
		}
		last = l.NewNo
	}
	return first, last
}

// revertHunk 把一块还原成改动前:在文件里把「新」文本换回「旧」文本。
func (a *App) revertHunk(f chFile, i int) {
	d := f.blocks[i]
	rel := a.wsRel(d.Path)
	cur, err := a.ide.fs.Read(rel)
	switch {
	case err != nil:
		a.toast = "无法读取 " + rel
		return
	case d.New == "" || !strings.Contains(cur, d.New):
		a.toast = "这一块已被改动过,无法自动撤销"
		return
	}
	if err := a.ide.fs.Write(rel, strings.Replace(cur, d.New, d.Old, 1)); err != nil {
		a.toast = "撤销失败:" + err.Error()
		return
	}
	a.rv.hunk[hunkKey(f.path, i)] = "rev"
	a.ide.reloadOpen()
	log.Info("撤销改动块", "path", rel, "hunk", i)
}

// wsRel 把改动里的路径换成相对项目根的路径(真实模式的工具调用带绝对路径)。
func (a *App) wsRel(p string) string {
	if a.multiRoot() {
		r, ok := a.wsRelMulti(p)
		if !ok && !strings.HasPrefix(p, "/") && len(a.repos) > 0 { // 相对路径:会话 cwd = 主文件夹
			return a.repos[0].name + "/" + strings.TrimPrefix(p, "./")
		}
		return r
	}
	if a.proj != "" {
		return strings.TrimPrefix(strings.TrimPrefix(p, a.proj), "/")
	}
	return p
}

// reviewView 是审查标签的内容。
func (a *App) reviewView(c *ui.Context) {
	t := a.tk
	a.rv.init()
	files := a.changedFiles()
	totalAdd, totalDel := 0, 0
	for _, f := range files {
		totalAdd += f.add
		totalDel += f.del
	}
	// 头:标题 / 统计 / 视图切换 / 批量操作
	ui.Row(c).Height(36).Shrink(0).PaddingX(12).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
		kit.Ic(c, "eye", kit.S12).TextColor(t.Ac)
		ui.Text(c, "本轮改动").FontWeight(700)
		count := itoa(len(files)) + " 个文件"
		if gs := a.reviewGroups(files); repoGroupCount(gs) > 1 {
			count += " · " + itoa(repoGroupCount(gs)) + " 个仓库"
		}
		ui.Text(c, count).TextColor(t.Tx3)
		ui.Text(c, "+"+itoa(totalAdd)).Font("monospace").FontSize(11.5).TextColor(t.Ok)
		ui.Text(c, "−"+itoa(totalDel)).Font("monospace").FontSize(11.5).TextColor(t.Err)
		ui.Spacer(c).Grow(1)
		if !a.wsNarrow() {
			on := 0
			if a.rv.side {
				on = 1
			}
			if i := kit.Seg(c, []kit.SegItem{{Text: "行内"}, {Text: "左右"}}, on, kit.SegOpt{Key: "rvseg"}); i >= 0 {
				a.rv.side = i == 1
			}
		}
		kit.Btn(c, "全部撤销", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "undo"}).OnClick(func() { a.reviewAll(files, "rev") })
		kit.Btn(c, "全部接受", kit.BtnOpt{Variant: kit.BtnSoft, Small: true, Icon: "check"}).OnClick(func() { a.reviewAll(files, "ok") })
		kit.Btn(c, "提交…", kit.BtnOpt{Variant: kit.BtnPri, Small: true}).OnClick(a.OpenSCM)
	})
	ui.Scroll(c).Key("review").Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Gap(8).Padding(10, 12, 12, 12).Children(func() {
			if len(files) == 0 {
				ui.Column(c).Padding(40, 0).AlignItems(ui.Center).Gap(4).Children(func() {
					ui.Text(c, "本轮还没有改动").TextColor(t.Tx2).FontWeight(500)
					ui.Text(c, "Claude 改了文件之后,会在这里逐块给你审查。").TextColor(t.Tx3).FontSize(theme.FsCap)
				})
				return
			}
			if gs := a.reviewGroups(files); gs != nil { // 多根工作区:按仓库分组
				if n := repoGroupCount(gs); n > 1 {
					a.reviewBanner(c, n)
				}
				first := true
				for _, g := range gs {
					a.reviewGroupHead(c, g)
					if a.rv.grpClosed["g"+itoa(g.idx)] {
						first = false
						continue
					}
					for _, f := range g.files {
						a.reviewFile(c, f, first)
						first = false
					}
				}
			} else {
				for i, f := range files {
					a.reviewFile(c, f, i == 0)
				}
			}
			ui.Row(c).Gap(6).AlignItems(ui.Center).Margin(4, 0, 0, 0).TextColor(t.Tx3).FontSize(theme.FsCap).Children(func() {
				kit.Ic(c, "info", kit.S12)
				ui.Text(c, "还没有提交。接受 = 保留(不写入 Git);撤销 = 把这一块还原成改动前。").Grow(1)
			})
		})
	})
	// 底部:已查看进度
	viewed := 0
	for _, f := range files {
		if a.rv.viewed[f.path] || a.fileDecided(f) {
			viewed++
		}
	}
	ui.Column(c).Shrink(0).Children(func() {
		a.reviewStageUndoBar(c)
		ui.Row(c).Height(28).PaddingX(12).Gap(8).AlignItems(ui.Center).BorderWidth(1, 0, 0, 0).BorderColor(t.Bd1).
			TextColor(t.Tx3).FontSize(theme.FsCap).Children(func() {
			ui.Text(c, a.reviewFooterText(files)).Grow(1).SingleLine()
			n, why := a.reviewStageAcceptedReason(files)
			kit.Btn(c, "暂存已接受的改动", kit.BtnOpt{Variant: kit.BtnSoft, Small: true, Icon: "plus", Disabled: n == 0, Tip: why}).OnClick(a.reviewStageAccepted)
			kit.Btn(c, "回退到这一回合之前", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "rewind"}).OnClick(a.reviewRewind)
			kit.Btn(c, "在源代码管理里暂存", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "branch"}).OnClick(a.OpenSCM)
		})
	})
}

// wsNarrow 工作区是否处于窄档(< 520:强制统一视图、隐藏次要控件)。
func (a *App) wsNarrow() bool { return a.wsW > 0 && a.wsW < 520 }

// reviewAll 批量处理所有块。
func (a *App) reviewAll(files []chFile, how string) {
	for _, f := range files {
		for i := range f.blocks {
			k := hunkKey(f.path, i)
			if a.rv.hunk[k] != "" {
				continue
			}
			if how == "rev" {
				a.revertHunk(f, i)
			} else {
				a.rv.hunk[k] = "ok"
			}
		}
	}
}

// reviewFile 画一个文件:可折叠的头 + 每块的操作条与差异行。
func (a *App) reviewFile(c *ui.Context, f chFile, first bool) {
	t := a.tk
	open, set := a.rv.open[f.path]
	if !set {
		open = first
	}
	closed := !open
	box := ui.Column(c).Key("rvf-"+f.path).Background(t.Raised).Radius(t.RMd()).Border(1, t.Bd1).Clip()
	box.Children(func() {
		head := ui.ButtonBase(c).Key("rvh-" + f.path).Height(30).PaddingX(10).Gap(6).AlignItems(ui.Center).Cursor(ui.CursorDefault).Transition(kit.Fade)
		if head.Hovered() {
			head.Background(t.Fill1)
		}
		head.OnClick(func() { a.rv.open[f.path] = closed })
		head.Children(func() {
			ch := kit.Ic(c, "chevD", kit.S12).TextColor(t.Tx3)
			ch.Rotate(ch.AnimateWith("rot", map[bool]float32{true: -90, false: 0}[closed], kit.DurBase, kit.EaseOut))
			kit.Ic(c, "code", kit.S12).TextColor(t.Tx3)
			ui.Text(c, baseName(f.path)).FontWeight(600).SingleLine().Shrink(0)
			ui.Text(c, a.fileDispDir(f.path)).FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine().Grow(1).Shrink(1)
			if f.add > 0 {
				ui.Text(c, "+"+itoa(f.add)).Font("monospace").FontSize(theme.FsCap).TextColor(t.Ok)
			}
			if f.del > 0 {
				ui.Text(c, "−"+itoa(f.del)).Font("monospace").FontSize(theme.FsCap).TextColor(t.Err)
			}
			// 「已查看」勾选
			v := a.rv.viewed[f.path] || a.fileDecided(f)
			vb := ui.ButtonBase(c).Key("rvv-" + f.path).Gap(5).AlignItems(ui.Center).TextColor(t.Tx3).FontSize(theme.FsCap).Shrink(0)
			vb.OnClick(func() { a.rv.viewed[f.path] = !a.rv.viewed[f.path] })
			vb.Children(func() {
				st := 0
				if v {
					st = 2
				}
				kit.Chk(c, st)
				ui.Text(c, "已查看")
			})
		})
		if closed {
			return
		}
		for i, d := range f.blocks {
			a.reviewHunk(c, f, i, d)
		}
	})
}

// reviewHunk 画一块:操作条(评论 / 撤销 / 接受)+ 差异行 + 评论框。
func (a *App) reviewHunk(c *ui.Context, f chFile, i int, d *core.DiffBlock) {
	t := a.tk
	k := hunkKey(f.path, i)
	st := a.rv.hunk[k]
	lo, hi := hunkRange(d)
	ls := diff.Lines(d.Old, d.New, d.Line, d.Line)
	ui.Row(c).Key("rvb-"+k).Height(24).PaddingX(10).Gap(8).AlignItems(ui.Center).Background(t.Fill1).
		BorderWidth(1, 0, 1, 0).BorderColor(t.Bd1).TextColor(t.Tx3).FontSize(theme.FsCap).Children(func() {
		lbl := "第 " + itoa(lo) + "–" + itoa(hi) + " 行"
		if lo == hi {
			lbl = "第 " + itoa(lo) + " 行"
		}
		ui.Text(c, lbl).Grow(1).SingleLine()
		switch st {
		case "ok":
			kit.Tag(c, "已接受", kit.KOk)
		case "rev":
			kit.Tag(c, "已撤销", kit.KErr)
		default:
			kit.Btn(c, "评论", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "chat"}).OnClick(func() {
				if a.rv.commenting == k {
					a.rv.commenting = ""
				} else {
					a.rv.commenting, a.rv.draft = k, a.rv.comments[k]
				}
			})
			kit.Btn(c, "撤销", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "undo"}).OnClick(func() { a.revertHunk(f, i) })
			kit.Btn(c, "接受", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "check"}).OnClick(func() { a.rv.hunk[k] = "ok" })
		}
	})
	ui.Column(c).Key("rvl-"+k).Padding(3, 0).Opacity(map[bool]float32{true: .38, false: 1}[st == "rev"]).Children(func() {
		if a.rv.side && !a.wsNarrow() {
			a.diffSplitLines(c, ls)
			return
		}
		for j, l := range ls {
			a.diffLine(c, j, l)
		}
	})
	if cm := a.rv.comments[k]; cm != "" && a.rv.commenting != k {
		ui.Row(c).Key("rvc-"+k).Margin(0, 10, 8, 10).Padding(6, 10).Gap(6).Background(t.AcSoft).Radius(t.RSm()).Children(func() {
			kit.Ic(c, "chat", kit.S12).TextColor(t.AcTx)
			ui.Column(c).Grow(1).Children(func() {
				ui.Text(c, cm).TextColor(t.Tx1)
				kit.Btn(c, "回复", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "reply"}).OnClick(func() {
					a.rv.commenting, a.rv.draft = k, cm+"\n"
				})
			})
		})
	}
	if a.rv.commenting == k {
		ui.Column(c).Key("rvi-"+k).Margin(2, 10, 8, 10).Padding(8).Gap(6).Background(t.BG).Radius(t.RMd()).Border(1, t.AcLine).Children(func() {
			ui.Text(c, "你 · 评论第 "+itoa(lo)+" 行").FontSize(theme.FsCap).TextColor(t.Tx3)
			ui.TextArea(c.Key("rvin-"+k), &a.rv.draft).Lines(1, 4).Placeholder("对这一块说点什么…").PlaceholderColor(t.Tx3).
				Background(t.Fill1).Radius(t.RSm()).BorderWidth(0).Padding(6, 8).FontSize(theme.FsBody).FocusRing(false)
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "评论只留在这里,点「发送给 Claude」才会作为下一条消息。").FontSize(theme.FsMicro).TextColor(t.Tx4).Grow(1)
				kit.Btn(c, "取消", kit.BtnOpt{Variant: kit.BtnGhost, Small: true}).OnClick(func() { a.rv.commenting = "" })
				kit.Btn(c, "发送给 Claude", kit.BtnOpt{Variant: kit.BtnPri, Small: true, Icon: "send"}).OnClick(func() {
					txt := trim(a.rv.draft)
					a.rv.commenting = ""
					if txt == "" {
						return
					}
					a.rv.comments[k] = txt
					a.submitKey("关于 "+f.path+" 第 "+itoa(lo)+" 行附近的改动:"+txt, false)
				})
			})
		})
	}
}

func trimSlash(s string) string {
	if n := len(s); n > 0 && s[n-1] == '/' {
		return s[:n-1]
	}
	return s
}

// splitRow 是左右对比的一行:左 = 旧侧,右 = 新侧,某一侧为 nil 表示这一侧留空。
type splitRow struct{ l, r *diff.Line }

// splitRows 把行内差异排成左右对比:相同行两侧都有;连续的删除与随后的新增一一配对,多出的一侧留空。
func splitRows(ls []diff.Line) []splitRow {
	var out []splitRow
	for i := 0; i < len(ls); {
		if ls[i].Kind == diff.Same {
			out = append(out, splitRow{&ls[i], &ls[i]})
			i++
			continue
		}
		var dels, adds []*diff.Line
		for ; i < len(ls) && ls[i].Kind == diff.Del; i++ {
			dels = append(dels, &ls[i])
		}
		for ; i < len(ls) && ls[i].Kind == diff.Add; i++ {
			adds = append(adds, &ls[i])
		}
		for k := 0; k < max(len(dels), len(adds)); k++ {
			var r splitRow
			if k < len(dels) {
				r.l = dels[k]
			}
			if k < len(adds) {
				r.r = adds[k]
			}
			out = append(out, r)
		}
	}
	return out
}

// diffSplitLines 画左右对比(两栏等宽,中间一条细线),窄档不会走到这里。
func (a *App) diffSplitLines(c *ui.Context, ls []diff.Line) {
	t := a.tk
	for i, r := range splitRows(ls) {
		ui.Row(c).Key(i).Height(monoLH).AlignItems(ui.Center).Children(func() {
			for side, ln := range []*diff.Line{r.l, r.r} {
				cell := ui.Row(c).Key(side).Grow(1).Basis(0).MinWidth(0).Height(monoLH).AlignItems(ui.Center).Clip()
				if side == 1 {
					cell.BorderWidth(0, 0, 0, 1).BorderColor(t.Bd1)
				}
				if ln != nil && ln.Kind == diff.Del {
					cell.Background(t.DiffDel)
				} else if ln != nil && ln.Kind == diff.Add {
					cell.Background(t.DiffAdd)
				}
				cell.Children(func() {
					if ln == nil {
						return
					}
					no := ln.NewNo
					if side == 0 {
						no = ln.OldNo
					}
					ui.Text(c, itoa(no)).Font("monospace").FontSize(11.5).TextColor(t.Tx4).TextAlign(ui.End).Width(30).PaddingEnd(6).Shrink(0)
					ui.RichText(c, kit.RustSpans(ln.Text, t.Tx1, 11.5)...).NoWrap().Grow(1).MinWidth(0).Clip()
				})
			}
		})
	}
}
