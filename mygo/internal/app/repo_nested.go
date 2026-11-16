package app

import (
	"github.com/ixxxxoooo/minicode/internal/theme"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// 嵌套仓库与子模块(设计稿 multi-git「嵌套仓库与子模块」):
// 每个工作区文件夹向下 2 层找 .git,找到的当作独立仓库挂在父文件夹下面;
// 一个文件属于离它最近的 .git 所在的仓库。

// pre 文件树路径前缀(没设置时退回 name,兼容旧构造)。
func (e *repoEntry) pre() string {
	if e.prefix == "" {
		return e.name
	}
	return e.prefix
}

// addNested 在第 parent 个文件夹下登记一个嵌套仓库,返回新的条目。
func (a *App) addNested(parent int, rel string, r vcs.Repo, kind string) *repoEntry {
	if kind == "" {
		kind = "嵌套"
	}
	root := a.repos[parent]
	e := &repoEntry{
		name: rel, prefix: root.pre() + "/" + rel, kind: kind, parent: parent,
		dir: root.dir + "/" + rel, repo: r, checked: true,
	}
	if r != nil {
		e.branch = r.Current()
	}
	a.repos = append(a.repos, e)
	return e
}

// repoIndexOf 文件树路径 → 所属仓库序号与仓库内相对路径;嵌套仓库(前缀更长)优先于外层文件夹。
func (a *App) repoIndexOf(p string) (int, string, bool) {
	best, bl := -1, -1
	for i, e := range a.repos {
		pre := e.pre()
		if (p == pre || strings.HasPrefix(p, pre+"/")) && len(pre) > bl {
			best, bl = i, len(pre)
		}
	}
	if best < 0 {
		return 0, p, false
	}
	return best, strings.TrimPrefix(strings.TrimPrefix(p, a.repos[best].pre()), "/"), true
}

// repoOrder 总览里的显示顺序:每个文件夹后面紧跟它下面的嵌套仓库。
func (a *App) repoOrder() []int {
	out := make([]int, 0, len(a.repos))
	for i, e := range a.repos {
		if e.kind == "" {
			out = append(out, i)
			for j, n := range a.repos {
				if n.kind != "" && n.parent == i {
					out = append(out, j)
				}
			}
		}
	}
	return out
}

// nestedAtTree 文件树里这一行是不是某个嵌套仓库的根目录(返回它的序号)。
func (a *App) nestedAtTree(p string) (int, bool) {
	for i, e := range a.repos {
		if e.kind != "" && e.pre() == p {
			return i, true
		}
	}
	return 0, false
}

// nestedBadges 文件树里嵌套仓库目录行右侧的徽标:类别标签 + 改动数。
func (a *App) nestedBadges(c *ui.Context, p string) {
	i, ok := a.nestedAtTree(p)
	if !ok {
		return
	}
	kit.Tag(c, a.repos[i].kind)
	if _, _, _, n := a.repoSummary(i); n > 0 {
		ui.Text(c, itoa(n)).FontSize(theme.Px(10)).FontWeight(600).TextColor(a.tk.Warn).FontFeatures("tnum").Shrink(0)
	}
}

// nestedSkip 发现嵌套仓库时不进的目录。
var nestedSkip = map[string]bool{"node_modules": true, "target": true, "vendor": true, "dist": true, "build": true, "__pycache__": true}

type nestedFound struct{ rel, kind string }

// findNested 在 dir 下 2 层内找 .git(目录或 worktree 文件),只读。
func findNested(dir string) []nestedFound {
	subs := gitmodulePaths(dir)
	var out []nestedFound
	var walk func(rel string, depth int)
	walk = func(rel string, depth int) {
		ents, err := os.ReadDir(filepath.Join(dir, rel))
		if err != nil {
			return
		}
		for _, en := range ents {
			nm := en.Name()
			if !en.IsDir() || nm == ".git" || strings.HasPrefix(nm, ".") || nestedSkip[nm] {
				continue
			}
			r := nm
			if rel != "" {
				r = rel + "/" + nm
			}
			if _, err := os.Stat(filepath.Join(dir, r, ".git")); err == nil {
				k := "嵌套"
				if subs[r] {
					k = "子模块"
				}
				out = append(out, nestedFound{r, k})
				continue // 仓库里面的不再往下找
			}
			if depth < 2 {
				walk(r, depth+1)
			}
		}
	}
	walk("", 1)
	sort.Slice(out, func(i, j int) bool { return out[i].rel < out[j].rel })
	return out
}

// gitmodulePaths 读 .gitmodules 里的 path = xxx。
func gitmodulePaths(dir string) map[string]bool {
	b, err := os.ReadFile(filepath.Join(dir, ".gitmodules"))
	if err != nil {
		return nil
	}
	m := map[string]bool{}
	for _, ln := range strings.Split(string(b), "\n") {
		ln = strings.TrimSpace(ln)
		if k, v, ok := strings.Cut(ln, "="); ok && strings.TrimSpace(k) == "path" {
			m[strings.Trim(strings.TrimSpace(v), "\"")] = true
		}
	}
	return m
}

// discoverNested 真实模式:后台为每个根文件夹找嵌套仓库并登记。失败只写日志,不影响其它仓库。
func (a *App) discoverNested() {
	if a.demoWs() || a.ws == nil {
		return
	}
	ws := a.ws
	for i, e := range a.repos {
		if e.kind != "" {
			continue
		}
		i, dir := i, e.dir
		a.bg("find-nested", func() func() {
			defer diag.SpanSlow("app", "查找嵌套仓库", 500*time.Millisecond)()
			found := findNested(dir)
			repos := make([]vcs.Repo, len(found))
			for k, f := range found {
				if cli, err := vcs.Open(filepath.Join(dir, f.rel)); err == nil {
					repos[k] = cli
				}
			}
			return func() {
				if a.ws != ws || i >= len(a.repos) || a.repos[i].dir != dir {
					return
				}
				for k, f := range found {
					if repos[k] == nil || a.hasNested(i, f.rel) {
						continue
					}
					a.addNested(i, f.rel, repos[k], f.kind)
					log.Info("发现嵌套仓库", "parent", a.repos[i].name, "rel", f.rel, "kind", f.kind)
				}
				a.syncTreeGit()
			}
		})
	}
}

func (a *App) hasNested(parent int, rel string) bool {
	for _, e := range a.repos {
		if e.kind != "" && e.parent == parent && e.name == rel {
			return true
		}
	}
	return false
}
