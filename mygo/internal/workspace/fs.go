// Package workspace 是 IDE 形态与后端共用的文件系统抽象:演示模式用内存里的沙盒项目,
// 真实模式读磁盘。所有「列目录 / 读文件 / 写文件」都经过它,界面不直接碰 os。
package workspace

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Entry 是目录里的一项。
type Entry struct {
	Name string
	Path string // 相对项目根,斜杠分隔
	Dir  bool
}

// FS 是项目文件系统。路径一律是相对项目根的斜杠路径("src/router.rs"),根自己是 ""。
type FS interface {
	Name() string // 项目名(根目录名)
	List(dir string) ([]Entry, error)
	Read(p string) (string, error)
	Write(p, content string) error
	Abs(p string) string // 绝对路径(演示项目是虚拟路径)
}

// sortEntries:文件夹在前,同类按名字(不区分大小写)。
func sortEntries(es []Entry) {
	sort.Slice(es, func(i, j int) bool {
		if es[i].Dir != es[j].Dir {
			return es[i].Dir
		}
		return strings.ToLower(es[i].Name) < strings.ToLower(es[j].Name)
	})
}

// ───────── 内存(演示) ─────────

// Mem 是内存里的项目:文件内容在 map 里,目录由路径推出。
type Mem struct {
	mu    sync.RWMutex
	name  string
	files map[string]string
	dirs  map[string]bool // 空目录
	// Prefer 指定根目录里这些名字优先排列(演示项目按设计稿的顺序摆放),其余按默认规则。
	Prefer []string
}

// NewMem 用文件表建一个内存项目。
func NewMem(name string, files map[string]string, emptyDirs ...string) *Mem {
	m := &Mem{name: name, files: map[string]string{}, dirs: map[string]bool{}}
	for k, v := range files {
		m.files[k] = v
	}
	for _, d := range emptyDirs {
		m.dirs[d] = true
	}
	return m
}

func (m *Mem) Name() string        { return m.name }
func (m *Mem) Abs(p string) string { return path.Join("/demo", m.name, p) }

func (m *Mem) List(dir string) ([]Entry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	prefix := ""
	if dir != "" {
		prefix = strings.TrimSuffix(dir, "/") + "/"
	}
	seen := map[string]bool{}
	var out []Entry
	add := func(rel string, isDir bool) {
		if !strings.HasPrefix(rel, prefix) {
			return
		}
		rest := rel[len(prefix):]
		if rest == "" {
			return
		}
		name, tail, nested := strings.Cut(rest, "/")
		if seen[name] {
			return
		}
		seen[name] = true
		out = append(out, Entry{Name: name, Path: prefix + name, Dir: nested && tail != "" || isDir})
	}
	for f := range m.files {
		add(f, false)
	}
	for d := range m.dirs {
		add(d, true)
	}
	if dir != "" && len(out) == 0 {
		if _, ok := m.dirs[dir]; !ok {
			return nil, os.ErrNotExist
		}
	}
	sortEntries(out)
	if dir == "" && len(m.Prefer) > 0 {
		rank := func(n string) int {
			for i, p := range m.Prefer {
				if p == n {
					return i
				}
			}
			return len(m.Prefer)
		}
		sort.SliceStable(out, func(i, j int) bool { return rank(out[i].Name) < rank(out[j].Name) })
	}
	return out, nil
}

func (m *Mem) Read(p string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.files[p]
	if !ok {
		return "", os.ErrNotExist
	}
	return s, nil
}

func (m *Mem) Write(p, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[p] = content
	return nil
}

// ───────── 磁盘 ─────────

// OS 是磁盘上的项目。
type OS struct{ root string }

// NewOS 以 root 为项目根。
func NewOS(root string) *OS { return &OS{root: filepath.Clean(root)} }

func (o *OS) Name() string        { return filepath.Base(o.root) }
func (o *OS) Abs(p string) string { return filepath.Join(o.root, filepath.FromSlash(p)) }

// 列目录时跳过的噪音目录(与旧版文件树的忽略规则一致)。
var ignored = map[string]bool{".git": true, "node_modules": true, "target": true, ".DS_Store": true, "dist": false}

func (o *OS) List(dir string) ([]Entry, error) {
	des, err := os.ReadDir(o.Abs(dir))
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, de := range des {
		n := de.Name()
		if ignored[n] {
			continue
		}
		out = append(out, Entry{Name: n, Path: path.Join(dir, n), Dir: de.IsDir()})
	}
	sortEntries(out)
	return out, nil
}

// 单文件读取上限:超过就拒绝(编辑器不是用来看大文件的)。
const MaxRead = 8 << 20

// ErrTooLarge 表示文件超过读取上限。
var ErrTooLarge = errors.New("文件过大")

func (o *OS) Read(p string) (string, error) {
	abs := o.Abs(p)
	st, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if st.Size() > MaxRead {
		return "", ErrTooLarge
	}
	b, err := os.ReadFile(abs)
	return string(b), err
}

// Write 原子写盘:先写临时文件再改名,中途崩溃不会留下半个文件。
func (o *OS) Write(p, content string) error {
	abs := o.Abs(p)
	tmp := abs + ".minicode-tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, abs)
}
