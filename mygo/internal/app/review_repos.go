package app

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 审查标签按仓库分组 + 「逐个提交」向导(设计稿「多仓库 Git」页)。
// Claude 一轮可能改了好几个仓库里的文件:审查只负责「看 / 接受 / 撤销」,提交仍在源代码管理里分别进行;
// 向导只是「依次切活动仓库 + 预填说明」,不替人按提交。

// 分组序号的特殊值。
const (
	grpNoRepo  = -1 // 文件夹不是 Git 仓库,或在任何文件夹之外
	grpOutside = -2
)

// reviewGroup 是审查标签里一个仓库(或「未纳入 Git」)的一组文件。
type reviewGroup struct {
	idx      int    // a.repos 的序号;grpOutside = 工作区之外
	name     string // 仓库 / 文件夹名
	branch   string
	hasRepo  bool
	files    []chFile
	add, del int
}

// repoIdxOfFile 改动文件所属的文件夹序号(多根时);单项目返回 grpNoRepo。
// 工具调用里的绝对路径按文件夹目录匹配;演示里的相对路径归第一个文件夹。
func (a *App) repoIdxOfFile(p string) int {
	if !a.multiRoot() {
		return grpNoRepo
	}
	best := -1
	for i, e := range a.repos { // 离文件最近的 .git:嵌套仓库的目录更长,优先
		if (p == e.dir || strings.HasPrefix(p, e.dir+"/")) && (best < 0 || len(e.dir) > len(a.repos[best].dir)) {
			best = i
		}
	}
	if best >= 0 {
		return best
	}
	if i, _, ok := a.repoIndexOf(p); ok {
		return i
	}
	if !strings.HasPrefix(p, "/") { // 相对路径:会话 cwd 默认是第一个文件夹
		return 0
	}
	return grpOutside
}

// reviewGroups 把改动文件按所属仓库分组(按首次出现顺序);单项目返回 nil。
func (a *App) reviewGroups(files []chFile) []reviewGroup {
	if !a.multiRoot() {
		return nil
	}
	var out []reviewGroup
	pos := map[int]int{}
	for _, f := range files {
		i := a.repoIdxOfFile(f.path)
		k, ok := pos[i]
		if !ok {
			g := reviewGroup{idx: i, name: "工作区之外"}
			if i >= 0 {
				e := a.repos[i]
				g.name, g.hasRepo = e.name, e.repo != nil
				g.branch, _, _, _ = a.repoSummary(i)
			}
			out = append(out, g)
			k = len(out) - 1
			pos[i] = k
		}
		out[k].files = append(out[k].files, f)
		out[k].add += f.add
		out[k].del += f.del
	}
	return out
}

// repoGroupCount 分组里真正是 Git 仓库的个数(决定要不要出「涉及多个仓库」的提示条)。
func repoGroupCount(gs []reviewGroup) int {
	n := 0
	for _, g := range gs {
		if g.hasRepo {
			n++
		}
	}
	return n
}

// reviewCommitTo 「提交到 xxx…」:切到该仓库、打开源代码管理、说明为空时预填草稿。
func (a *App) reviewCommitTo(idx int) {
	if !a.multiRoot() || idx < 0 || idx >= len(a.repos) || a.repos[idx].repo == nil {
		return
	}
	e := a.repos[idx]
	if idx == a.repoAct {
		if strings.TrimSpace(a.sc.msg) == "" {
			cs := e.changes
			if a.sc.loaded {
				cs = a.sc.changes
			}
			a.sc.msg = draftCommitMsg(cs)
		}
	} else {
		if strings.TrimSpace(e.msg) == "" {
			e.msg = draftCommitMsg(e.changes)
		}
		a.setActiveRepo(idx)
	}
	a.sc.commitErr = ""
	log.Info("审查 → 提交到仓库", "repo", e.name)
	a.OpenSCM()
}

// ───────── 逐个提交向导 ─────────

// commitWizard 是「逐个提交」向导的状态。
type commitWizard struct {
	order    []int          // 要处理的仓库序号(有改动的仓库)
	pos      int            // 当前第几个
	res      map[int]string // 仓库 → ok 已提交 / skip 已跳过
	finished time.Time      // 全部处理完的时刻(之后 6 秒淡出)
	filled   map[int]bool   // 已经预填过说明的仓库(状态读完才预填一次,用户清空后不再覆盖)
}

// startCommitWizard 从审查分组里取出有未提交改动的仓库,依次提交。
func (a *App) startCommitWizard() {
	gs := a.reviewGroups(a.changedFiles())
	var order []int
	for _, g := range gs {
		if !g.hasRepo {
			continue
		}
		if _, _, _, n := a.repoSummary(g.idx); n > 0 {
			order = append(order, g.idx)
		}
	}
	if len(order) == 0 {
		a.toast = "这些仓库都没有未提交的改动"
		return
	}
	a.cw = &commitWizard{order: order, res: map[int]string{}, filled: map[int]bool{}}
	log.Info("开始逐个提交", "repos", len(order))
	a.wizardGo(0)
}

// wizardGo 切到第 pos 个仓库并预填说明。
func (a *App) wizardGo(pos int) {
	w := a.cw
	if w == nil || pos < 0 || pos >= len(w.order) {
		return
	}
	w.pos = pos
	a.reviewCommitTo(w.order[pos])
}

// wizardNext 记录当前仓库的结果并前进;最后一个处理完则结束。
func (a *App) wizardNext(res string) {
	w := a.cw
	if w == nil || !w.finished.IsZero() {
		return
	}
	w.res[w.order[w.pos]] = res
	if w.pos+1 < len(w.order) {
		a.wizardGo(w.pos + 1)
		return
	}
	w.finished = time.Now()
	log.Info("逐个提交结束", "committed", w.count("ok"), "skipped", w.count("skip"))
}

// wizardAfterCommit 活动仓库提交成功后调用:向导在进行且就是这个仓库,自动前进。
func (a *App) wizardAfterCommit() {
	w := a.cw
	if w != nil && w.finished.IsZero() && a.repoAct == w.order[w.pos] {
		a.wizardNext("ok")
	}
}

func (w *commitWizard) count(res string) int {
	n := 0
	for _, r := range w.res {
		if r == res {
			n++
		}
	}
	return n
}

// wizardBar 画在源代码管理提交框上方的向导条(没有向导时什么都不画)。
func (a *App) wizardBar(c *ui.Context) {
	w := a.cw
	if w == nil {
		return
	}
	t := a.tk
	if !w.finished.IsZero() {
		if time.Since(w.finished) > 6*time.Second {
			a.cw = nil
			return
		}
		a.frameSoon()
		ui.Row(c).Shrink(0).Gap(6).AlignItems(ui.Center).Padding(7, 8).Margin(2, 2, 6, 2).Radius(t.RSm()).Background(t.OkSoft).Children(func() {
			kit.Ic(c, "check", kit.S12).TextColor(t.Ok)
			msg := "已提交 " + itoa(w.count("ok")) + " 个仓库"
			if names := w.skippedNames(a); len(names) > 0 {
				msg += " · 跳过 " + strings.Join(names, "、")
			}
			ui.Text(c, msg).Grow(1).FontWeight(500).SingleLine()
			kit.Btn(c, "查看图谱", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "fork"}).OnClick(a.openGraph)
		})
		return
	}
	cur := w.order[w.pos]
	if cur != a.repoAct { // 手动切走了:暂停,不是退出
		ui.Row(c).Shrink(0).Gap(6).AlignItems(ui.Center).Padding(7, 8).Margin(2, 2, 6, 2).Radius(t.RSm()).Background(t.Fill1).Children(func() {
			kit.Ic(c, "info", kit.S12).TextColor(t.Tx3)
			ui.Text(c, "逐个提交已暂停").Grow(1).TextColor(t.Tx3).SingleLine()
			kit.Btn(c, "回到向导", kit.BtnOpt{Variant: kit.BtnGhost, Small: true}).OnClick(func() { a.wizardGo(w.pos) })
			kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "退出向导"}).OnClick(func() { a.cw = nil })
		})
		return
	}
	if !w.filled[cur] && a.sc.loaded { // 活动仓库的状态读完后补预填(真实模式刚切过去时还没有改动列表)
		w.filled[cur] = true
		if strings.TrimSpace(a.sc.msg) == "" {
			a.sc.msg = draftCommitMsg(a.sc.changes)
		}
	}
	name := a.repos[cur].name
	sub := name + " · " + a.branch
	if a.sc.commitErr == "" {
		if st, _ := a.sc.split(); a.sc.loaded && len(st) == 0 {
			sub += " · 先在「更改」里暂存要提交的文件"
		} else if strings.TrimSpace(a.sc.msg) != "" {
			sub += " · 已预填说明,检查后 ⌘↩ 提交"
		}
	}
	// 窄栏(导航栏)里放得下:第一行 标题 + 进度 + 跳过 + 退出,第二行说明 / 失败行
	ui.Column(c).Shrink(0).Gap(3).Padding(6, 8).Margin(2, 2, 6, 2).Radius(t.RSm()).Background(t.AcSoft).Border(1, t.Ac).Children(func() {
		ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
			kit.Ic(c, "commit", kit.S12).TextColor(t.Ac)
			ui.Text(c, "逐个提交").FontWeight(500).SingleLine().Shrink(0)
			ui.Text(c, itoa(w.pos+1)+" / "+itoa(len(w.order))).TextColor(t.Tx3).FontSize(theme.FsCap).FontFeatures("tnum").Shrink(0)
			ui.Row(c).Gap(3).Shrink(0).Children(func() {
				for i := range w.order {
					dot := t.Bd2
					switch {
					case i < w.pos:
						dot = t.Ok
					case i == w.pos:
						dot = t.Ac
					}
					ui.Box(c).Size(6, 6).Radius(3).Background(dot)
				}
			})
			ui.Spacer(c).Grow(1)
			if a.sc.commitErr == "" {
				kit.Btn(c, "跳过这个仓库", kit.BtnOpt{Variant: kit.BtnGhost, Small: true}).OnClick(func() { a.wizardNext("skip") })
			}
			kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "退出向导"}).OnClick(func() { a.cw = nil })
		})
		if a.sc.commitErr != "" {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Padding(4, 2).Radius(t.RSm()).Background(t.ErrSoft).Children(func() {
				kit.Ic(c, "alert", kit.S12).TextColor(t.Err)
				ui.Text(c, name+" · "+a.sc.commitErr).Grow(1).FontSize(theme.FsCap).TextColor(t.Err).MaxLines(2)
				kit.Btn(c, "重试", kit.BtnOpt{Variant: kit.BtnGhost, Small: true}).OnClick(func() { a.scmCommit(false) })
				kit.Btn(c, "跳过这个仓库", kit.BtnOpt{Variant: kit.BtnGhost, Small: true}).OnClick(func() {
					a.sc.commitErr = ""
					a.wizardNext("skip")
				})
			})
		} else {
			ui.Text(c, sub).FontSize(theme.FsMicro).TextColor(t.Tx3).MaxLines(2)
		}
	})
}

func (w *commitWizard) skippedNames(a *App) []string {
	var out []string
	for _, ri := range w.order {
		if w.res[ri] == "skip" && ri < len(a.repos) {
			out = append(out, a.repos[ri].name)
		}
	}
	return out
}

// ───────── 审查标签的分组渲染 ─────────

// reviewBanner 涉及多个仓库时顶部的提示条。
func (a *App) reviewBanner(c *ui.Context, n int) {
	t := a.tk
	ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(7, 10).Radius(t.RMd()).Background(t.AcSoft).Children(func() {
		kit.Ic(c, "info", kit.S12).TextColor(t.Ac)
		ui.Text(c, "本轮改动涉及 "+itoa(n)+" 个仓库,提交需要分别进行。").Grow(1)
		kit.Btn(c, "逐个提交", kit.BtnOpt{Variant: kit.BtnPri, Small: true, Icon: "commit"}).OnClick(a.startCommitWizard)
	})
}

// reviewGroupHead 一个分组的头:折叠箭头 + 仓库名 + 分支 + 统计 + 「提交到 xxx…」。
func (a *App) reviewGroupHead(c *ui.Context, g reviewGroup) {
	t := a.tk
	key := "g" + itoa(g.idx)
	closed := a.rv.grpClosed[key]
	h := ui.ButtonBase(c).Key("rvg-" + key).Height(theme.Px(32)).PaddingX(10).Gap(8).AlignItems(ui.Center).Radius(t.RMd()).Background(t.Fill1).Cursor(ui.CursorDefault).Transition(kit.Fade)
	if h.Hovered() {
		h.Background(t.Fill2)
	}
	h.OnClick(func() { a.rv.grpClosed[key] = !closed })
	h.Children(func() {
		ch := kit.Ic(c, "chevD", kit.S12).TextColor(t.Tx3)
		ch.Rotate(ch.AnimateWith("rot", map[bool]float32{true: -90, false: 0}[closed], kit.DurBase, kit.EaseOut))
		icon := "branch"
		if !g.hasRepo {
			icon = "folder"
		}
		kit.Ic(c, icon, kit.S12).TextColor(t.Tx3)
		ui.Text(c, g.name).FontWeight(500).SingleLine().Shrink(0)
		if g.hasRepo && g.branch != "" {
			ui.Text(c, g.branch).TextColor(t.Tx3).SingleLine().Shrink(1).MinWidth(0)
		}
		if !g.hasRepo {
			kit.Tag(c, "未纳入 Git", kit.KNeutral)
		}
		ui.Text(c, itoa(len(g.files))+" 个文件").FontSize(theme.FsCap).TextColor(t.Tx4).Shrink(0)
		ui.Text(c, "+"+itoa(g.add)).Font(theme.CodeFont()).FontSize(theme.FsMicro).TextColor(t.Ok).Shrink(0)
		ui.Text(c, "−"+itoa(g.del)).Font(theme.CodeFont()).FontSize(theme.FsMicro).TextColor(t.Err).Shrink(0)
		ui.Spacer(c).Grow(1)
		if g.hasRepo {
			idx := g.idx
			kit.Btn(c, "提交到 "+g.name+"…", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "commit"}).OnClick(func() { a.reviewCommitTo(idx) })
		} else if g.idx >= 0 {
			idx := g.idx
			kit.Btn(c, "初始化仓库", kit.BtnOpt{Variant: kit.BtnGhost, Small: true}).OnClick(func() { a.initRepo(idx) })
		}
	})
}

// demoRepoFiles 演示工作区:除了第一个文件夹里的改动,再让 web-dashboard(有仓库)与 design-assets(非仓库)
// 各有一轮改动,审查标签才有「多个仓库 + 未纳入 Git」可看。
func (a *App) demoRepoFiles() []chFile {
	if !a.demoWs() || !a.multiRoot() {
		return nil
	}
	var out []chFile
	add := func(folder, rel, status string) {
		for _, e := range a.repos {
			if e.name != folder {
				continue
			}
			cur, err := a.ide.fs.Read(e.name + "/" + rel)
			if err != nil {
				return
			}
			old := ""
			if e.repo != nil {
				old, _ = e.repo.Show("HEAD", rel)
			}
			if status == "A" {
				old = ""
			}
			p := e.dir + "/" + rel
			blk := &core.DiffBlock{Path: p, Old: old, New: cur, Line: 1}
			ad, de := reviewStat(blk)
			out = append(out, chFile{path: p, status: status, add: ad, del: de, blocks: []*core.DiffBlock{blk}})
			return
		}
	}
	add("web-dashboard", "src/charts.ts", "M")
	add("web-dashboard", "src/components/Card.tsx", "A")
	add("design-assets", "palette.md", "A")
	return out
}

// fileDispDir 审查文件头里显示的目录:多根时去掉文件夹的绝对前缀,只留仓库内目录。
func (a *App) fileDispDir(p string) string {
	p = a.shortPath(p) // 单项目:绝对路径先换成相对项目根,标题里不再露出 /private/tmp/… 这类长前缀
	if i := a.repoIdxOfFile(p); i >= 0 {
		p = strings.TrimPrefix(p, a.repos[i].dir+"/")
	}
	return trimSlash(dirOf(p))
}
