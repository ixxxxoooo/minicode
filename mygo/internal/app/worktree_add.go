package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 设置 → 源代码管理 → Worktree 的「新建」:输入分支名,在存放位置下建 <项目名>-<分支名> 目录。
// 分支已存在就直接检出,不存在则基于当前 HEAD 新建。

// worktreeBase 解析「存放位置」偏好(支持 ~ 开头),空则用默认 ~/.minicode/worktrees。
func worktreeBase(pref string) string {
	pref = strings.TrimSpace(pref)
	if pref == "" {
		pref = "~/.minicode/worktrees"
	}
	if pref == "~" || strings.HasPrefix(pref, "~/") {
		if h, err := os.UserHomeDir(); err == nil {
			pref = filepath.Join(h, strings.TrimPrefix(pref, "~"))
		}
	}
	return pref
}

// worktreeDirName 把项目名与分支名拼成目录名(分支里的 / 等字符换成 -)。
func worktreeDirName(proj, branch string) string {
	r := strings.NewReplacer("/", "-", "\\", "-", " ", "-", ":", "-")
	return filepath.Base(proj) + "-" + r.Replace(strings.TrimSpace(branch))
}

// worktreeAdd 在 repo 下新建 worktree,返回目录。分支名不合法、目录已存在、git 失败都给中文原因。
func worktreeAdd(repo, base, branch string) (string, error) {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return "", fmt.Errorf("先写一个分支名,例如 feat/login")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	run := func(args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0")
		return cmd.CombinedOutput()
	}
	if _, err := run("check-ref-format", "--branch", branch); err != nil {
		return "", fmt.Errorf("分支名不合法:不能含空格、~ ^ : ? * [ \\ 或以 / 开头结尾")
	}
	dir := filepath.Join(base, worktreeDirName(repo, branch))
	if _, err := os.Stat(dir); err == nil {
		return "", fmt.Errorf("目录已存在:%s(换个分支名,或先删掉它)", dir)
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", fmt.Errorf("无法创建存放位置 %s:%v", base, err)
	}
	var args []string
	if _, err := run("show-ref", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		args = []string{"worktree", "add", dir, branch} // 分支已存在:直接检出
	} else {
		args = []string{"worktree", "add", "-b", branch, dir}
	}
	if out, err := run(args...); err != nil {
		return "", fmt.Errorf("%s", friendlyGitErr(string(out)))
	}
	return dir, nil
}

// worktreeNewRow 是 Worktree 区块底部的「新建」行:分支名输入 + 创建。
func (a *App) worktreeNewRow(c *ui.Context) {
	t := a.tk
	canWhy := ""
	switch {
	case a.demoWs():
		canWhy = "演示模式不会真的创建目录"
	case a.repo == nil || a.proj == "":
		canWhy = "当前项目不是 Git 仓库"
	}
	rowBox(a, c, false, 44).Children(func() {
		kit.Ic(c, "plus", kit.S12).TextColor(t.Tx3)
		ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
			kit.Field(c, "wt-new", kit.FieldOpt{Err: a.set.wtErr}, func(box ui.Element) {
				box.BorderWidth(0).Background(ui.Transparent).PaddingX(0).Height(theme.Px(24)).Shrink(0)
				in := ui.TextInputBase(c.Key("wt-new-in"), &a.set.wtNew).Grow(1).FontSize(theme.FsCap + .5).Font(theme.CodeFont()).Placeholder("新分支名(已存在则直接检出),例如 feat/login")
				in.OnChange(func() { a.set.wtErr = "" })
				in.OnSubmit(a.worktreeCreate)
			})
		})
		kit.Btn(c, "新建 worktree", kit.BtnOpt{Small: true, Variant: kit.BtnSoft, Disabled: canWhy != "", Loading: a.set.wtBuilding, Tip: canWhy}).OnClick(a.worktreeCreate)
	})
}

func (a *App) worktreeCreate() {
	if a.set.wtBuilding || a.repo == nil || a.demoWs() {
		return
	}
	branch := strings.TrimSpace(a.set.wtNew)
	if branch == "" {
		a.set.wtErr = "先写一个分支名,例如 feat/login"
		a.shakeField("wt-new")
		return
	}
	a.set.wtBuilding = true
	repo, base := a.proj, worktreeBase(a.prefs.Str(pScmWorktrees, ""))
	setLog.Info("新建 worktree", "repo", repo, "branch", branch)
	a.bg("worktree-add", func() func() {
		end := diag.SpanSlow("app", "git worktree add", 5*time.Second, "repo", repo)
		dir, err := worktreeAdd(repo, base, branch)
		end()
		return func() {
			a.set.wtBuilding = false
			if err != nil {
				a.set.wtErr = err.Error()
				a.shakeField("wt-new")
				setLog.Warn("worktree 新建失败", "branch", branch, "err", err)
				return
			}
			a.set.wtNew, a.set.wtErr = "", ""
			a.set.wtAt = time.Time{} // 列表立即重读
			a.toast = "已新建 worktree:" + dir
			setLog.Info("worktree 已新建", "branch", branch, "dir", dir)
		}
	})
}
