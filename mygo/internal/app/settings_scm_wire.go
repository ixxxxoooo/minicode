package app

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// pScmWorktrees 是 worktree 存放位置偏好(与 settings_tabs.go 里的键一致)
const pScmWorktrees = "scm.worktrees"

type worktreeRow struct {
	branch, path string
}

// ScmFetchTick 按偏好 scm.fetch 在后台 git fetch(由 main 定时调用)。
func (a *App) ScmFetchTick() {
	if a == nil || a.prefs == nil || !a.prefs.Bool(pScmFetch, false) {
		return
	}
	if a.repo == nil {
		return
	}
	a.bg("scm-fetch", func() func() {
		end := diag.SpanSlow("app", "git fetch", 60*time.Second, "repo", a.proj)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, "git", "-C", a.proj, "fetch", "--quiet")
		cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0")
		err := cmd.Run()
		end()
		if err != nil {
			setLog.Warn("自动 fetch 失败", "err", err)
		}
		return nil
	})
}

func (a *App) scmWorktreeBody(c *ui.Context) {
	a.scmWorktreeList(c)
	a.worktreeNewRow(c)
}

func (a *App) scmWorktreeList(c *ui.Context) {
	t := a.tk
	rows := a.worktreeCached()
	if len(rows) == 0 {
		rowBox(a, c, true, 44).Children(func() {
			msg := "没有 worktree 或不是 Git 仓库"
			if a.set.wtAt.IsZero() && !a.demoWs() {
				msg = "正在读取…"
			}
			ui.Text(c, msg).FontSize(theme.FsCap).TextColor(t.Tx3).Grow(1)
		})
		return
	}
	for i, w := range rows {
		w := w
		rowBox(a, c, i == 0, 40).Children(func() {
			kit.Ic(c, "branch", kit.S12).TextColor(t.Tx3)
			ui.Column(c).Gap(2).Grow(1).Children(func() {
				ui.Text(c, w.branch).FontWeight(500)
				ui.Text(c, w.path).Font("monospace").FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
			})
			if i == 0 {
				// 第一项是主工作树,不能删除。
				ui.Text(c, "主工作树").FontSize(theme.FsCap).TextColor(t.Tx4)
				return
			}
			kit.Ib(c, "trash", kit.IbOpt{Small: true, Tip: "删除"}).OnClick(func() {
				a.confirmDlg("删除 worktree", "将删除 "+w.path+" 并移除注册。", "删除", true, func() { a.removeWorktree(w.path) })
			})
		})
	}
}

// worktreeCached 返回缓存的 worktree 列表;过期(5 秒)或从未读过时在后台刷新,
// 渲染线程不跑 git。
func (a *App) worktreeCached() []worktreeRow {
	if a.demoWs() {
		return a.worktreeList()
	}
	if !a.set.wtBusy && time.Since(a.set.wtAt) > 5*time.Second {
		a.set.wtBusy = true
		a.bg("worktree-list", func() func() {
			end := diag.SpanSlow("app", "git worktree list", 2*time.Second, "repo", a.proj)
			rows := a.worktreeList()
			end()
			return func() {
				a.set.wt = rows
				a.set.wtAt = time.Now()
				a.set.wtBusy = false
			}
		})
	}
	return a.set.wt
}

func (a *App) worktreeList() []worktreeRow {
	if a.demoWs() {
		var rows []worktreeRow
		for _, w := range demo.DemoWorktrees() {
			rows = append(rows, worktreeRow{branch: w.Branch, path: w.Path})
		}
		return rows
	}
	if a.repo == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", "-C", a.proj, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return nil
	}
	var rows []worktreeRow
	var cur worktreeRow
	for _, ln := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(ln, "worktree ") {
			if cur.path != "" {
				rows = append(rows, cur)
			}
			cur = worktreeRow{path: strings.TrimPrefix(ln, "worktree ")}
		} else if strings.HasPrefix(ln, "branch ") {
			cur.branch = strings.TrimPrefix(ln, "branch refs/heads/")
		}
	}
	if cur.path != "" {
		rows = append(rows, cur)
	}
	return rows
}

func (a *App) removeWorktree(path string) {
	a.bg("worktree-rm", func() func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "git", "-C", a.proj, "worktree", "remove", path)
		out, err := cmd.CombinedOutput()
		return func() {
			if err != nil {
				msg := friendlyWorktreeRmErr(string(out))
				a.toast = "删除失败:" + msg
				setLog.Warn("worktree 删除失败", "err", err, "out", string(out))
				return
			}
			a.toast = "已删除 worktree"
			a.set.wtAt = time.Time{} // 让列表立即重读
		}
	})
}

// friendlyWorktreeRmErr 把 git worktree remove 的报错翻成一句话(有未提交改动 / 被锁定 / 其它)。
func friendlyWorktreeRmErr(raw string) string {
	m := strings.ToLower(raw)
	switch {
	case strings.Contains(m, "contains modified or untracked files"):
		return "里面有未提交的改动,先提交或丢弃它们"
	case strings.Contains(m, "locked"):
		return "这个 worktree 被锁定了(git worktree unlock 后再删)"
	case strings.Contains(m, "is a main working tree"):
		return "主工作树不能删除"
	}
	return friendlyGitErr(raw)
}
