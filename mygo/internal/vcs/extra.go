package vcs

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// 块级暂存、文件历史、追溯、储藏、提交上的操作。接口见 Repo。

// BlameLine 是追溯里的一行(下标 = 行号 - 1)。
type BlameLine struct {
	Hash, Author, When, Summary string
	Uncommitted                 bool
}

// Stash 是一条储藏。
type Stash struct {
	Ref, Msg, When string
	Files          int
	held           []Change // 演示仓库:储藏时收走的改动,应用时放回
}

// ───────── CLI ─────────

// gitIn 同 gitRaw,另给标准输入。
func (r *CLI) gitIn(timeout time.Duration, stdin string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"--no-optional-locks"}, args...)...)
	cmd.Dir = r.Dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
	cmd.Stdin = strings.NewReader(stdin)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return out.String(), nil
}

func (r *CLI) Index(p string) (string, bool) {
	out, err := r.gitRaw(5*time.Second, "show", ":"+p)
	return out, err == nil
}

// StageContent 把 content 写成 p 在暂存区里的内容(块级暂存 / 取消暂存的落点)。
func (r *CLI) StageContent(p, content string) error {
	hash, err := r.gitIn(10*time.Second, content, "hash-object", "-w", "--stdin")
	if err != nil {
		return err
	}
	mode := "100644"
	if out, err := r.git(5*time.Second, "ls-files", "-s", "--", p); err == nil {
		if f := strings.Fields(out); len(f) > 0 {
			mode = f[0]
		}
	}
	_, err = r.git(10*time.Second, "update-index", "--add", "--cacheinfo", mode+","+strings.TrimSpace(hash)+","+p)
	return err
}

func (r *CLI) FileLog(p string, n int) ([]Commit, error) {
	out, err := r.git(10*time.Second, "log", "--follow", "-n", strconv.Itoa(n),
		"--format=%H"+logSep+"%P"+logSep+"%an"+logSep+"%ar"+logSep+"%s"+logSep+"%D", "--", p)
	if err != nil {
		return nil, nil // 没有提交 / 未跟踪:没有历史
	}
	return parseLog(out), nil
}

func (r *CLI) Blame(p string) ([]BlameLine, error) {
	out, err := r.gitRaw(15*time.Second, "blame", "--porcelain", "-w", "--", p)
	if err != nil {
		return nil, err
	}
	return parseBlame(out), nil
}

// parseBlame 解析 porcelain:每行一个头 "<哈希> <原行> <新行> [<组行数>]",首次出现的提交带作者 / 时间 / 摘要,正文行以 Tab 开头。
func parseBlame(out string) []BlameLine {
	info := map[string]BlameLine{}
	var lines []BlameLine
	var cur BlameLine
	for _, ln := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(ln, "\t"):
			lines = append(lines, cur)
		case strings.HasPrefix(ln, "author "):
			cur.Author = strings.TrimPrefix(ln, "author ")
		case strings.HasPrefix(ln, "author-time "):
			if sec, err := strconv.ParseInt(strings.TrimPrefix(ln, "author-time "), 10, 64); err == nil {
				cur.When = zhRelative(relativeFrom(time.Unix(sec, 0)))
			}
		case strings.HasPrefix(ln, "summary "):
			cur.Summary = strings.TrimPrefix(ln, "summary ")
			info[cur.Hash] = cur
		default:
			f := strings.Fields(ln)
			if len(f) >= 3 && len(f[0]) >= 40 {
				if known, ok := info[f[0]]; ok {
					cur = known
				} else {
					cur = BlameLine{Hash: f[0], Uncommitted: strings.Trim(f[0], "0") == ""}
				}
			}
		}
	}
	for i := range lines {
		if lines[i].Uncommitted {
			lines[i].Author, lines[i].When, lines[i].Summary = "你", "未提交", "未提交的改动"
		}
	}
	return lines
}

// relativeFrom 把时间换成 git 风格的英文相对时间("3 hours ago"),交给 zhRelative 翻成中文。
func relativeFrom(t time.Time) string {
	d := time.Since(t)
	unit := func(n int, u string) string {
		if n != 1 {
			u += "s"
		}
		return fmt.Sprintf("%d %s ago", n, u)
	}
	switch {
	case d < time.Minute:
		return unit(int(d.Seconds()), "second")
	case d < time.Hour:
		return unit(int(d.Minutes()), "minute")
	case d < 24*time.Hour:
		return unit(int(d.Hours()), "hour")
	case d < 30*24*time.Hour:
		return unit(int(d.Hours()/24), "day")
	case d < 365*24*time.Hour:
		return unit(int(d.Hours()/24/30), "month")
	}
	return unit(int(d.Hours()/24/365), "year")
}

func (r *CLI) Stashes() ([]Stash, error) {
	out, err := r.git(5*time.Second, "stash", "list", "--format=%gd\x1f%gs\x1f%cr")
	if err != nil {
		return nil, err
	}
	var ss []Stash
	for _, ln := range strings.Split(out, "\n") {
		f := strings.Split(ln, "\x1f")
		if len(f) < 3 {
			continue
		}
		s := Stash{Ref: f[0], Msg: f[1], When: zhRelative(f[2])}
		if names, err := r.git(5*time.Second, "stash", "show", "--name-only", f[0]); err == nil && names != "" {
			s.Files = len(strings.Split(names, "\n"))
		}
		ss = append(ss, s)
	}
	return ss, nil
}

func (r *CLI) StashPush(msg string) error {
	args := []string{"stash", "push", "-u"}
	if msg != "" {
		args = append(args, "-m", msg)
	}
	_, err := r.git(30*time.Second, args...)
	return err
}

func (r *CLI) StashApply(i int, drop bool) error {
	verb := "apply"
	if drop {
		verb = "pop"
	}
	_, err := r.git(30*time.Second, "stash", verb, "stash@{"+strconv.Itoa(i)+"}")
	return err
}

func (r *CLI) StashDrop(i int) error {
	_, err := r.git(10*time.Second, "stash", "drop", "stash@{"+strconv.Itoa(i)+"}")
	return err
}

func (r *CLI) Checkout(hash string) error {
	_, err := r.git(30*time.Second, "checkout", "--detach", hash)
	return err
}

func (r *CLI) CherryPick(hash string) error {
	_, err := r.git(30*time.Second, "cherry-pick", hash)
	return err
}

func (r *CLI) RevertCommit(hash string) error {
	_, err := r.git(30*time.Second, "revert", "--no-edit", hash)
	return err
}

func (r *CLI) CreateAt(name, hash string) error {
	if !ValidName(name) {
		return ErrInvalidName
	}
	_, err := r.git(15*time.Second, "switch", "-c", name, hash)
	return err
}

func (r *CLI) UndoCommit() error {
	_, err := r.git(15*time.Second, "reset", "--soft", "HEAD~1")
	return err
}

func (r *CLI) ResetKeep(hash string) error {
	if !hashRe.MatchString(hash) {
		return ErrInvalidHash
	}
	_, err := r.git(30*time.Second, "reset", "--keep", hash)
	return err
}

// ───────── Mem ─────────

func (m *Mem) Index(p string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.index[p]; ok {
		return s, true
	}
	s, ok := m.head[p]
	return s, ok
}

func (m *Mem) StageContent(p, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.index == nil {
		m.index = map[string]string{}
	}
	m.index[p] = content
	for _, c := range m.changes {
		if c.Path == p && c.Staged {
			return nil
		}
	}
	m.changes = append(m.changes, Change{Path: p, Status: 'M', Staged: true})
	return nil
}

// FileLog 演示:文件出现在某次提交的文件清单里就算;一个都没有时给设计稿里的四条。
func (m *Mem) FileLog(p string, n int) ([]Commit, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Commit
	for _, c := range m.log {
		for _, f := range m.files[c.Hash] {
			if f.Path == p {
				out = append(out, c)
				break
			}
		}
	}
	if len(out) > n {
		out = out[:n]
	}
	return out, nil
}

// Blame 演示:没有历史,按「提交里碰过它的最近提交」给前几行,其余行归最早的提交,有未提交改动的行归「你 · 未提交」。
func (m *Mem) Blame(p string) ([]BlameLine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return nil, fmt.Errorf("演示仓库没有追溯数据")
}

func (m *Mem) Stashes() ([]Stash, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Stash(nil), m.stash...), nil
}

func (m *Mem) StashPush(msg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if msg == "" {
		msg = "WIP on " + m.cur
	}
	held := append([]Change(nil), m.changes...)
	m.changes = nil
	m.stash = append([]Stash{{Ref: "stash@{0}", Msg: msg, When: "刚刚", Files: len(held), held: held}}, m.stash...)
	for i := range m.stash {
		m.stash[i].Ref = fmt.Sprintf("stash@{%d}", i)
	}
	return nil
}

func (m *Mem) StashApply(i int, drop bool) error {
	return m.stashTake(i, drop, true)
}

func (m *Mem) stashTake(i int, drop, restore bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if i < 0 || i >= len(m.stash) {
		return fmt.Errorf("没有这条储藏")
	}
	if restore {
		m.changes = append(m.changes, m.stash[i].held...)
	}
	if drop {
		m.stash = append(m.stash[:i:i], m.stash[i+1:]...)
	}
	return nil
}

func (m *Mem) StashDrop(i int) error { return m.stashTake(i, true, false) }

func (m *Mem) Checkout(hash string) error     { return nil }
func (m *Mem) CherryPick(hash string) error   { return nil }
func (m *Mem) RevertCommit(hash string) error { return nil }
func (m *Mem) CreateAt(name, hash string) error {
	return m.Create(name)
}

// SetStash 演示用:灌入储藏列表。
func (m *Mem) SetStash(s []Stash) { m.mu.Lock(); m.stash = s; m.mu.Unlock() }

func (m *Mem) UndoCommit() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.log) == 0 {
		return fmt.Errorf("没有可撤销的提交")
	}
	top := m.log[0]
next:
	for _, f := range m.files[top.Hash] {
		f.Staged = true
		for _, c := range m.changes {
			if c.Path == f.Path && c.Staged {
				continue next // 已有暂存项:不重复列出
			}
		}
		m.changes = append(m.changes, f)
	}
	m.log = m.log[1:]
	if len(m.log) > 0 {
		m.log[0].Refs = append([]string{"HEAD", m.cur}, m.log[0].Refs...)
	}
	if m.ahead > 0 {
		m.ahead--
	}
	return nil
}

func (m *Mem) ResetKeep(hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	at := -1
	for i, c := range m.log {
		if c.Hash == hash {
			at = i
			break
		}
	}
	if at < 0 {
		return fmt.Errorf("当前分支上找不到这个提交")
	}
	if at == 0 {
		return nil
	}
	old := m.log[0].Refs
	m.log = m.log[at:]
	var keep []string
	for _, r := range old { // 旧 HEAD 上的引用(HEAD / 当前分支)搬到新 HEAD,其余随旧提交消失
		if r == "HEAD" || r == m.cur {
			keep = append(keep, r)
		}
	}
	m.log[0].Refs = append(keep, m.log[0].Refs...)
	m.ahead = max(m.ahead-at, 0)
	return nil
}
