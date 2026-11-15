package app

import (
	"sort"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diff"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 检查器(Agent 右栏)的标签。
var inspTabs = []struct{ id, name string }{
	{"changes", "改动"}, {"files", "文件"}, {"term", "终端"}, {"plan", "计划"}, {"tasks", "任务"},
}

// chFile 是「改动」里的一个文件。
type chFile struct {
	path     string
	status   string // M / A
	add, del int
	block    *core.DiffBlock // 第一处差异(默认展开用)
}

// changedFiles 汇总会话里 Claude 改过的文件(按首次出现顺序)。
func (a *App) changedFiles() []chFile {
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
						out = append(out, chFile{path: d.Path, status: st, block: d})
						i = len(out) - 1
					}
					out[i].add += ad
					out[i].del += de
				}
			case *core.SubagentCard:
				visit(v.Children)
			}
		}
	}
	visit(a.st.Items)
	return out
}

// inspector 是 Agent 右栏:标签条 + 内容。宽度 304,收起时宽度与外边距一起过渡到 0。
func (a *App) inspector(c *ui.Context, gap float32) {
	t := a.tk
	show := a.inspOpen && a.mode == ModeAgent
	w, mg := float32(theme.WInsp), gap
	if !show {
		w, mg = 0, 0
	}
	p := a.pane(c).Key("insp").Width(w).Shrink(0).Margin(0, 0, 0, mg).ClipX().Transition(morph)
	if !show {
		p.Opacity(0)
	}
	p.Children(func() {
		ui.Column(c).Width(theme.WInsp).Grow(1).MinHeight(0).Children(func() {
			files := a.changedFiles()
			items := make([]kit.TabItem, len(inspTabs))
			for i, tb := range inspTabs {
				items[i] = kit.TabItem{ID: tb.id, Text: tb.name}
				switch tb.id {
				case "changes":
					items[i].Count = len(files) + a.extraChanges()
				case "tasks":
					items[i].Count = a.taskCount()
				}
			}
			if id := kit.Tabs(c, items, a.inspTab, theme.HTab, func() {
				kit.Ib(c, "plus", kit.IbOpt{Small: true}).Margin(0, 2, 0, 0)
			}); id != "" {
				a.inspTab = id
			}
			ui.Column(c).Grow(1).MinHeight(0).Padding(6, 0).Clip().Children(func() {
				switch a.inspTab {
				case "changes":
					a.changesTab(c, files)
				case "tasks":
					a.tasksTab(c)
				default:
					ui.Text(c, "").Margin(8)
				}
			})
		})
	})
	_ = t
}

// extraChanges 演示模式里工作区还有一个没被本会话碰过的未提交文件(case_test.rs),与设计稿的「3 个文件」一致。
func (a *App) extraChanges() int {
	if a.demoExtra {
		return 1
	}
	return 0
}

func (a *App) taskCount() int {
	n := 0
	for _, tk := range a.st.Tasks {
		if tk.Visible() && tk.Status == core.TaskRunning {
			n++
		}
	}
	return n
}

func (a *App) changesTab(c *ui.Context, files []chFile) {
	t := a.tk
	totalAdd, totalDel := 0, 0
	for _, f := range files {
		totalAdd += f.add
		totalDel += f.del
	}
	extra := a.extraChanges()
	if extra > 0 {
		totalAdd += 2
	}
	ui.Row(c).Gap(6).Padding(0, 12, 6, 12).AlignItems(ui.Center).Shrink(0).Children(func() {
		ui.Text(c, itoa(len(files)+extra)+" 个文件").TextColor(t.Tx3)
		ui.Text(c, "+"+itoa(totalAdd)).Font("monospace").FontSize(11.5).TextColor(t.Ok)
		ui.Text(c, "−"+itoa(totalDel)).Font("monospace").FontSize(11.5).TextColor(t.Err)
		ui.Spacer(c).Grow(1)
		kit.Btn(c, "审查", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "eye"})
		kit.Btn(c, "提交", kit.BtnOpt{Variant: kit.BtnSoft, Small: true, Icon: "check"})
	})
	ui.Scroll(c).Key("ch").Grow(1).Children(func() {
		ui.Column(c).Margin(0, 6).Children(func() {
			for i, f := range files {
				a.changeRow(c, f.path, f.status, f.add, f.del, i == 0)
				if i == 0 && f.block != nil {
					ui.Column(c).Margin(3, 0, 6, 0).Children(func() { a.diffCard(c, f.block, false) })
				}
			}
			if extra > 0 {
				a.changeRow(c, "tests/case_test.rs", "A", 2, 0, false)
			}
		})
	})
}

func (a *App) changeRow(c *ui.Context, path, status string, add, del int, on bool) {
	t := a.tk
	r := kit.RowEl(c, on).Key("ch-" + path)
	r.Children(func() {
		kit.Ic(c, "code", kit.S12)
		ui.Text(c, baseName(path)).SingleLine().Shrink(0)
		ui.Text(c, trimSlash(dirOf(path))).FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine().Grow(1).Shrink(1)
		if add > 0 {
			ui.Text(c, "+"+itoa(add)).Font("monospace").FontSize(theme.FsMicro).TextColor(t.Ok)
		}
		if del > 0 {
			ui.Text(c, "−"+itoa(del)).Font("monospace").FontSize(theme.FsMicro).TextColor(t.Err)
		}
		col := t.Warn
		if status == "A" {
			col = t.Ok
		}
		ui.Row(c).Width(10).Justify(ui.Center).Children(func() {
			ui.Text(c, status).FontSize(theme.FsMicro).FontWeight(500).TextColor(col)
		})
	})
}

func trimSlash(s string) string {
	if n := len(s); n > 0 && s[n-1] == '/' {
		return s[:n-1]
	}
	return s
}

// tasksTab 是「任务」标签:运行中的在前,已结束的收在小标题下。
func (a *App) tasksTab(c *ui.Context) {
	t := a.tk
	var run, done []*core.BackgroundTask
	for _, tk := range a.st.Tasks {
		if !tk.Visible() {
			continue
		}
		if tk.Status == core.TaskRunning {
			run = append(run, tk)
		} else {
			done = append(done, tk)
		}
	}
	sort.SliceStable(run, func(i, j int) bool { return run[i].StartedMs < run[j].StartedMs })
	ui.Column(c).Margin(0, 6).Children(func() {
		for _, tk := range run {
			ui.Row(c).Key("tk-"+tk.TaskID).Padding(5, 8).Gap(6).AlignItems(ui.Start).Children(func() {
				ui.Row(c).Margin(3, 0, 0, 0).Children(func() { kit.Spin(c, t.Ac) })
				ui.Column(c).Grow(1).MinWidth(0).Children(func() {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Text(c, tk.Description).FontWeight(500).TextColor(t.Tx1).SingleLine().Shrink(1)
						kit.Tag(c, tk.Kind.Label())
					})
					if tk.Summary != "" {
						ui.Text(c, tk.Summary).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
					}
				})
				ui.Text(c, core.FormatElapsed(a.st.Now()-tk.StartedMs)).FontSize(theme.FsMicro).TextColor(t.Tx4).FontFeatures("tnum")
			})
		}
		if len(done) > 0 {
			kit.Grp(c, "已结束").Margin(6, 0, 0, 0)
			for _, tk := range done {
				kit.RowEl(c, false).Key("tkd-" + tk.TaskID).Children(func() {
					kit.Ic(c, "check", kit.S12).TextColor(t.Ok)
					ui.Text(c, tk.Description).SingleLine().Grow(1)
					ui.Text(c, core.FormatElapsed(tk.EndedMs-tk.StartedMs)).FontSize(theme.FsMicro).TextColor(t.Tx4).FontFeatures("tnum")
				})
			}
		}
	})
}
