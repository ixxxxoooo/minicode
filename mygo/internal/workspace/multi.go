package workspace

import (
	"errors"
	"strings"
)

// Root 是多根工作区里的一个文件夹。
type Root struct {
	Name string // 顶层显示名(重名时自动加后缀)
	FS   FS
}

// Multi 把多个文件夹拼成一棵树:顶层是各文件夹(名字即第一段路径),往下是各自的内容。
// 路径形如 "web-dashboard/src/app.ts";第一段决定落在哪个根。
type Multi struct {
	name  string
	roots []Root
}

// NewMulti 用名字和各根建多根文件系统;重名的根加 " (2)" 之类的后缀,保证第一段唯一。
func NewMulti(name string, roots []Root) *Multi {
	m := &Multi{name: name}
	used := map[string]int{}
	for _, r := range roots {
		n := r.Name
		if n == "" {
			n = r.FS.Name()
		}
		used[n]++
		if used[n] > 1 {
			n = n + " (" + itoa(used[n]) + ")"
		}
		m.roots = append(m.roots, Root{Name: n, FS: r.FS})
	}
	return m
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

// Roots 各根(顺序 = 添加顺序,第一个是主文件夹)。
func (m *Multi) Roots() []Root { return m.roots }

func (m *Multi) Name() string { return m.name }

// Split 把多根路径拆成「根下标、根内相对路径」;路径为空或找不到根时 ok=false。
func (m *Multi) Split(p string) (int, string, bool) {
	p = strings.Trim(p, "/")
	if p == "" {
		return -1, "", false
	}
	first, rest, _ := strings.Cut(p, "/")
	for i, r := range m.roots {
		if r.Name == first {
			return i, rest, true
		}
	}
	return -1, "", false
}

func (m *Multi) List(dir string) ([]Entry, error) {
	if strings.Trim(dir, "/") == "" {
		out := make([]Entry, 0, len(m.roots))
		for _, r := range m.roots {
			out = append(out, Entry{Name: r.Name, Path: r.Name, Dir: true})
		}
		return out, nil // 顶层保持添加顺序(主文件夹在最前),不排序
	}
	i, rest, ok := m.Split(dir)
	if !ok {
		return nil, errors.New("不在工作区里:" + dir)
	}
	es, err := m.roots[i].FS.List(rest)
	for k := range es {
		es[k].Path = m.roots[i].Name + "/" + es[k].Path
	}
	return es, err
}

func (m *Multi) Read(p string) (string, error) {
	i, rest, ok := m.Split(p)
	if !ok {
		return "", errors.New("不在工作区里:" + p)
	}
	return m.roots[i].FS.Read(rest)
}

func (m *Multi) Write(p, content string) error {
	i, rest, ok := m.Split(p)
	if !ok {
		return errors.New("不在工作区里:" + p)
	}
	return m.roots[i].FS.Write(rest, content)
}

func (m *Multi) Abs(p string) string {
	if i, rest, ok := m.Split(p); ok {
		return m.roots[i].FS.Abs(rest)
	}
	if strings.Trim(p, "/") == "" && len(m.roots) > 0 {
		return m.roots[0].FS.Abs("")
	}
	return p
}
