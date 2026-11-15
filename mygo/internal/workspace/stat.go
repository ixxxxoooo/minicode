package workspace

import (
	"os"
	"strings"
)

// IsDir 判断相对路径在 fs 里是不是目录(文件不存在时按「能列出子项」推断)。
func IsDir(fs FS, p string) bool {
	p = strings.Trim(p, "/")
	if p == "" {
		return true
	}
	if m, ok := fs.(*Multi); ok {
		if _, rest, ok2 := m.Split(p); ok2 {
			if rest == "" {
				return true
			}
			i, inner, ok3 := m.Split(p)
			if !ok3 {
				return false
			}
			return IsDir(m.roots[i].FS, inner)
		}
		return false
	}
	if mem, ok := fs.(*Mem); ok {
		return mem.isDir(p)
	}
	st, err := os.Stat(fs.Abs(p))
	return err == nil && st.IsDir()
}

func (m *Mem) isDir(p string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.dirs[p] {
		return true
	}
	prefix := p + "/"
	for f := range m.files {
		if strings.HasPrefix(f, prefix) || f == p {
			if f == p {
				return false
			}
			return true
		}
	}
	for d := range m.dirs {
		if strings.HasPrefix(d, prefix) || d == p {
			return true
		}
	}
	_, err := m.Read(p)
	return err != nil
}
