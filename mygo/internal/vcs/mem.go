package vcs

import (
	"fmt"
	"sync"
)

// Mem 是内存里的假仓库(演示 / 测试用)。
type Mem struct {
	mu  sync.Mutex
	cur string
	br  []Branch
}

// NewMem 造一个带设计稿里那几个分支的演示仓库。
func NewMem() *Mem {
	return &Mem{cur: "main", br: []Branch{
		{Name: "main", When: "当前"},
		{Name: "feat/auth", When: "2 小时前"},
		{Name: "feat/ui-redesign", When: "刚刚"},
		{Name: "origin/dev", Remote: true, When: "昨天"},
	}}
}

func (m *Mem) Current() string { m.mu.Lock(); defer m.mu.Unlock(); return m.cur }

func (m *Mem) Branches() ([]Branch, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Branch, len(m.br))
	for i, b := range m.br {
		b.Current = !b.Remote && b.Name == m.cur
		out[i] = b
	}
	return out, nil
}

func (m *Mem) Switch(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, b := range m.br {
		if b.Name != name {
			continue
		}
		if b.Remote { // 远程分支:建立同名本地分支
			local := name[len("origin/"):]
			m.br = append(m.br, Branch{Name: local, When: "刚刚"})
			name = local
		}
		m.cur = name
		return nil
	}
	return fmt.Errorf("没有分支 %q", name)
}

func (m *Mem) Create(name string) error {
	if !ValidName(name) {
		return ErrInvalidName
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, b := range m.br {
		if !b.Remote && b.Name == name {
			return fmt.Errorf("分支 %q 已存在", name)
		}
	}
	m.br = append(m.br, Branch{Name: name, When: "刚刚"})
	m.cur = name
	return nil
}
