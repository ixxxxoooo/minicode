package vcs

import (
	"strconv"
	"strings"
	"time"
)

// parseNameStatus 解析 git diff --name-status 输出。
func parseNameStatus(out string) []Change {
	var cs []Change
	for _, ln := range strings.Split(out, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		parts := strings.SplitN(ln, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		st := parts[0]
		p := parts[1]
		if i := strings.Index(p, " -> "); i >= 0 {
			p = p[i+4:]
		}
		c := Change{Path: p, Staged: false}
		switch {
		case strings.HasPrefix(st, "A"):
			c.Status = 'A'
		case strings.HasPrefix(st, "D"):
			c.Status = 'D'
		default:
			c.Status = 'M'
		}
		cs = append(cs, c)
	}
	return cs
}

func (r *CLI) CompareCommits(a, b string) ([]Change, error) {
	out, err := r.git(20*time.Second, "diff", "--name-status", a, b)
	if err != nil {
		return nil, err
	}
	return parseNameStatus(out), nil
}

func (r *CLI) StashFiles(i int) ([]Change, error) {
	ref := "stash@{" + strconv.Itoa(i) + "}"
	out, err := r.git(10*time.Second, "stash", "show", "--name-status", ref)
	if err != nil {
		return nil, err
	}
	return parseNameStatus(out), nil
}

func (r *CLI) StashFileContent(i int, path string) (old, nw string, ok bool) {
	ref := "stash@{" + strconv.Itoa(i) + "}"
	parent := ref + "^"
	nw, ok = r.Show(ref, path)
	if !ok {
		return "", "", false
	}
	old, _ = r.Show(parent, path)
	return old, nw, true
}
