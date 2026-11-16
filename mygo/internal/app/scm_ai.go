package app

import (
	"fmt"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/diff"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// SetTextGenerator 注入一次性文本生成器(真实模式 = claude -p 小模型);没有时提交说明只走本地启发式。
func (a *App) SetTextGenerator(f func(dir, prompt string) (string, error)) { a.genText = f }

// commitPrompt 把待提交的改动整理成给模型的提示:文件清单 + 每个文件的差异节选(总量封顶)。
func (a *App) commitPrompt(cs []vcs.Change) string {
	var b strings.Builder
	b.WriteString("你是 Git 提交说明助手。根据下面的改动写一条提交说明:\n")
	b.WriteString("- 用中文 Conventional Commits 格式:type(scope): 描述;type 取 feat / fix / refactor / docs / test / chore / perf / style。\n")
	b.WriteString("- 第一行不超过 50 个汉字,不加句号;改动较大时空一行后用 2-4 条短横线列要点。\n")
	b.WriteString("- 只输出提交说明本身,不要解释、不要代码围栏、不要引号。\n\n改动清单:\n")
	left := 6000 // 差异节选的总字符预算
	for i, c := range cs {
		if i >= 30 {
			fmt.Fprintf(&b, "…另有 %d 个文件\n", len(cs)-i)
			break
		}
		fmt.Fprintf(&b, "- %c %s (+%d −%d)\n", c.Status, c.Path, c.Add, c.Del)
	}
	b.WriteString("\n差异节选:\n")
	for _, c := range cs {
		if left <= 0 {
			break
		}
		old, _ := a.repo.Show("HEAD", c.Path)
		var now string
		if c.Staged {
			now, _ = a.repo.Index(c.Path)
		} else if s, err := a.ide.fs.Read(c.Path); err == nil {
			now = s
		}
		if c.Status == 'D' {
			now = ""
		}
		n := 0
		fmt.Fprintf(&b, "### %s\n", c.Path)
		for _, l := range diff.Lines(old, now, 1, 1) {
			if l.Kind == diff.Same || n >= 40 || left <= 0 {
				continue
			}
			sign := "+"
			if l.Kind == diff.Del {
				sign = "-"
			}
			line := sign + l.Text
			if r := []rune(line); len(r) > 160 {
				line = string(r[:160]) + "…"
			}
			b.WriteString(line + "\n")
			n++
			left -= len(line)
		}
	}
	return b.String()
}

// cleanCommitMsg 去掉模型常带的围栏、引号与「提交说明:」前缀,压缩多余空行。
func cleanCommitMsg(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		if i := strings.Index(s, "\n"); i >= 0 {
			s = s[i+1:]
		}
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	}
	s = strings.TrimSpace(s)
	for _, p := range []string{"提交说明:", "提交说明：", "Commit message:", "commit message:"} {
		s = strings.TrimSpace(strings.TrimPrefix(s, p))
	}
	s = strings.Trim(s, "\"'“”`")
	var out []string
	blank := false
	for _, ln := range strings.Split(s, "\n") {
		ln = strings.TrimRight(ln, " \t")
		if ln == "" {
			if blank || len(out) == 0 {
				continue
			}
			blank = true
		} else {
			blank = false
		}
		out = append(out, ln)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// scmGenMsg 起草提交说明:开了「AI 提交说明」且有生成器就用模型(后台、可看到进度),失败或没开退回本地启发式。
// scmCancelGenMsg 取消正在进行的 AI 提交说明生成。
func (a *App) scmCancelGenMsg() {
	s := &a.sc
	if !s.genning {
		return
	}
	s.genGen++
	s.genning = false
	a.toast = "已取消生成提交说明"
}

func (a *App) scmGenMsg() {
	s := &a.sc
	if s.genning {
		return
	}
	staged, work := s.split()
	pick := staged
	if len(pick) == 0 {
		pick = work
	}
	if len(pick) == 0 || a.repo == nil {
		a.toast = "没有可描述的更改"
		return
	}
	fallback := draftCommitMsg(pick)
	if a.genText == nil || !a.prefs.Bool("scm.aiMsg", true) {
		s.msg = fallback
		return
	}
	prompt, dir, gen := a.commitPrompt(pick), a.proj, a.genText
	s.genning = true
	s.genGen++
	my := s.genGen
	a.bg("scm.aimsg", func() func() {
		defer diag.Recover("生成提交说明")
		out, err := gen(dir, prompt)
		msg := cleanCommitMsg(out)
		return func() {
			if my != s.genGen {
				return
			}
			s.genning = false
			if err != nil || msg == "" {
				diag.L("app").Warn("AI 提交说明失败,改用本地草稿", "err", err)
				s.msg = fallback
				a.toast = "AI 生成失败,已改用本地草稿"
				return
			}
			s.msg = msg
		}
	})
}
