package vcs

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Status 解析 `git status --porcelain=v1 -z -uall`:每项 "XY 路径\0",重命名后面再跟一项旧路径。
// X 是暂存区状态,Y 是工作区状态;同一文件两边都有改动时给出两条。
func (r *CLI) Status() ([]Change, error) {
	out, err := r.git(10*time.Second, "status", "--porcelain=v1", "-z", "-uall")
	if err != nil {
		return nil, err
	}
	cs := parseStatus(out)
	// 行数统计:暂存区与工作区各取一次 numstat
	staged := r.numstat("diff", "--cached", "--numstat", "-z")
	work := r.numstat("diff", "--numstat", "-z")
	for i := range cs {
		c := &cs[i]
		src := work
		if c.Staged {
			src = staged
		}
		if n, ok := src[c.Path]; ok {
			c.Add, c.Del = n[0], n[1]
		} else if c.Status == 'U' || (c.Status == 'A' && !c.Staged) {
			c.Add = r.countLines(c.Path)
		}
	}
	return cs, nil
}

func parseStatus(out string) []Change {
	var cs []Change
	parts := strings.Split(out, "\x00")
	for i := 0; i < len(parts); i++ {
		e := parts[i]
		if len(e) < 4 {
			continue
		}
		x, y, p := e[0], e[1], e[3:]
		if x == 'U' || y == 'U' || (x == y && (x == 'A' || x == 'D')) { // 合并冲突:UU / AA / DD / AU / UA / DU / UD
			cs = append(cs, Change{Path: p, Status: StatusConflict})
			continue
		}
		if x == 'R' || x == 'C' { // 重命名 / 复制:下一项是旧路径
			i++
		}
		if x == '?' && y == '?' {
			cs = append(cs, Change{Path: p, Status: 'U'})
			continue
		}
		if x == '!' {
			continue
		}
		if x != ' ' {
			cs = append(cs, Change{Path: p, Status: x, Staged: true})
		}
		if y != ' ' {
			cs = append(cs, Change{Path: p, Status: y})
		}
	}
	return cs
}

// numstat 解析 `git diff --numstat -z`:"增\t删\t路径\0"(重命名是 "增\t删\t\0旧\0新\0")。
func (r *CLI) numstat(args ...string) map[string][2]int {
	out, err := r.git(10*time.Second, args...)
	m := map[string][2]int{}
	if err != nil {
		return m
	}
	parts := strings.Split(out, "\x00")
	for i := 0; i < len(parts); i++ {
		f := strings.SplitN(parts[i], "\t", 3)
		if len(f) < 3 {
			continue
		}
		a, _ := strconv.Atoi(f[0]) // 二进制文件是 "-",按 0 算
		d, _ := strconv.Atoi(f[1])
		p := f[2]
		if p == "" && i+2 < len(parts) { // 重命名
			p = parts[i+2]
			i += 2
		}
		m[p] = [2]int{a, d}
	}
	return m
}

func (r *CLI) countLines(p string) int {
	b, err := os.ReadFile(filepath.Join(r.Dir, p))
	if err != nil || len(b) > 1<<20 {
		return 0
	}
	n := strings.Count(string(b), "\n")
	if len(b) > 0 && b[len(b)-1] != '\n' {
		n++
	}
	return n
}

func (r *CLI) Stage(paths ...string) error {
	_, err := r.git(15*time.Second, append([]string{"add", "--"}, paths...)...)
	return err
}

func (r *CLI) Unstage(paths ...string) error {
	if _, err := r.git(5*time.Second, "rev-parse", "--verify", "-q", "HEAD"); err != nil { // 还没有首个提交
		_, err = r.git(15*time.Second, append([]string{"rm", "--cached", "-q", "--"}, paths...)...)
		return err
	}
	_, err := r.git(15*time.Second, append([]string{"restore", "--staged", "--"}, paths...)...)
	return err
}

func (r *CLI) Discard(p string) error {
	// 未跟踪:直接删除;已跟踪:还原到暂存区版本
	if _, err := r.git(5*time.Second, "ls-files", "--error-unmatch", "--", p); err != nil {
		return os.Remove(filepath.Join(r.Dir, p))
	}
	_, err := r.git(15*time.Second, "restore", "--", p)
	return err
}

func (r *CLI) Commit(msg string, amend bool) error {
	if strings.TrimSpace(msg) == "" && !amend {
		return fmt.Errorf("提交说明不能为空")
	}
	args := []string{"commit", "-q"}
	if amend {
		args = append(args, "--amend")
	}
	if strings.TrimSpace(msg) != "" {
		args = append(args, "-m", msg)
	} else {
		args = append(args, "--no-edit")
	}
	_, err := r.git(30*time.Second, args...)
	return err
}

const logSep = "\x1f"

func (r *CLI) Log(n int) ([]Commit, error) {
	out, err := r.git(10*time.Second, "log", "--all", "--date-order", "-n", strconv.Itoa(n),
		"--format=%H"+logSep+"%P"+logSep+"%an"+logSep+"%ar"+logSep+"%s"+logSep+"%D")
	if err != nil {
		if strings.Contains(err.Error(), "does not have any commits") {
			return nil, nil
		}
		return nil, err
	}
	return parseLog(out), nil
}

func parseLog(out string) []Commit {
	var cs []Commit
	for _, ln := range strings.Split(out, "\n") {
		f := strings.Split(ln, logSep)
		if len(f) < 6 {
			continue
		}
		c := Commit{Hash: f[0], Author: f[2], When: zhRelative(f[3]), Subject: f[4]}
		if f[1] != "" {
			c.Parents = strings.Fields(f[1])
		}
		for _, ref := range strings.Split(f[5], ", ") {
			ref = strings.TrimSpace(ref)
			switch {
			case ref == "":
			case strings.HasPrefix(ref, "HEAD -> "):
				c.Refs = append(c.Refs, "HEAD", strings.TrimPrefix(ref, "HEAD -> "))
			case strings.HasSuffix(ref, "/HEAD"):
			default:
				c.Refs = append(c.Refs, strings.TrimPrefix(ref, "tag: "))
			}
		}
		cs = append(cs, c)
	}
	return cs
}

func (r *CLI) CommitFiles(hash string) ([]Change, error) {
	ns, err := r.git(10*time.Second, "show", "--name-status", "--format=", "-z", hash)
	if err != nil {
		return nil, err
	}
	stats := r.numstat("show", "--numstat", "--format=", "-z", hash)
	var cs []Change
	parts := strings.Split(ns, "\x00")
	for i := 0; i+1 < len(parts); i += 2 {
		st, p := parts[i], parts[i+1]
		if st == "" {
			continue
		}
		if st[0] == 'R' || st[0] == 'C' { // 重命名:状态后面跟 旧路径、新路径
			if i+2 < len(parts) {
				p = parts[i+2]
				i++
			}
		}
		c := Change{Path: p, Status: st[0], Staged: true}
		if n, ok := stats[p]; ok {
			c.Add, c.Del = n[0], n[1]
		}
		cs = append(cs, c)
	}
	return cs, nil
}

func (r *CLI) Show(rev, p string) (string, bool) {
	out, err := r.gitRaw(5*time.Second, "show", rev+":"+p)
	return out, err == nil
}

func (r *CLI) AheadBehind() (int, int) {
	out, err := r.git(5*time.Second, "rev-list", "--left-right", "--count", "@{u}...HEAD")
	if err != nil {
		return 0, 0
	}
	var behind, ahead int
	fmt.Sscanf(out, "%d\t%d", &behind, &ahead)
	return ahead, behind
}

func (r *CLI) Pull() error {
	_, err := r.git(60*time.Second, "pull", "--ff-only", "-q")
	return err
}

func (r *CLI) Push() error {
	_, err := r.git(60*time.Second, "push", "-q")
	return err
}

func (r *CLI) PushForce() error {
	_, err := r.git(60*time.Second, "push", "--force-with-lease", "-q")
	return err
}
