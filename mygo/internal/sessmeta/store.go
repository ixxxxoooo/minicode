// Package sessmeta 保存会话在本机的整理信息:自定义标题、置顶、归档。
// 只记 id → 信息,不改 Claude 的会话文件;写盘是原子的(临时文件 + 改名),损坏的文件改名留底。
package sessmeta

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

var log = diag.L("sessmeta")

// Meta 是一个会话的整理信息。
type Meta struct {
	Title    string `json:"title,omitempty"` // 自定义标题;空 = 用 Claude 生成的
	Pinned   bool   `json:"pinned,omitempty"`
	Archived bool   `json:"archived,omitempty"`
}

func (m Meta) zero() bool { return m == Meta{} }

// Store 是整理信息的存储(path 为空 = 只在内存里,演示用)。
type Store struct {
	mu   sync.Mutex
	path string
	m    map[string]Meta
}

// NewMem 只在内存里的存储。
func NewMem() *Store { return &Store{m: map[string]Meta{}} }

// Open 打开磁盘上的存储;读不到就是空,损坏则改名留底后从空开始。
func Open(path string) *Store {
	s := &Store{path: path, m: map[string]Meta{}}
	b, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	if err := json.Unmarshal(b, &s.m); err != nil || s.m == nil {
		log.Warn("会话整理信息损坏,已改名留底并从空开始", "err", err)
		_ = os.Rename(path, path+".bad-"+time.Now().Format("20060102-150405"))
		s.m = map[string]Meta{}
	}
	return s
}

// Get 取一个会话的整理信息(没有记录 = 零值)。
func (s *Store) Get(id string) Meta {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.m[id]
}

// Persistent 是否落盘。
func (s *Store) Persistent() bool { return s.path != "" }

func (s *Store) update(id string, f func(m *Meta)) error {
	if id == "" || strings.HasPrefix(id, "new:") {
		return nil // 新会话还没有 id
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	m := s.m[id]
	f(&m)
	if m.zero() {
		delete(s.m, id)
	} else {
		s.m[id] = m
	}
	return s.save()
}

// SetTitle 设置自定义标题;空串 = 恢复原标题。
func (s *Store) SetTitle(id, title string) error {
	title = strings.TrimSpace(title)
	return s.update(id, func(m *Meta) { m.Title = title })
}

// SetPinned 置顶 / 取消置顶。
func (s *Store) SetPinned(id string, v bool) error {
	return s.update(id, func(m *Meta) { m.Pinned = v })
}

// SetArchived 归档 / 取消归档。
func (s *Store) SetArchived(id string, v bool) error {
	return s.update(id, func(m *Meta) { m.Archived = v })
}

// Forget 会话被删除时清掉它的记录。
func (s *Store) Forget(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.m[id]; !ok {
		return nil
	}
	delete(s.m, id)
	return s.save()
}

// Len 记录条数。
func (s *Store) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.m)
}

func (s *Store) save() error {
	if s.path == "" {
		return nil
	}
	b, err := json.MarshalIndent(s.m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
