// Package prefs 是用户偏好的存取:一个扁平的「键 → 值」表,落在 ~/.minicode/settings.json。
//
// 设置页的每个控件对应一个键(形如 "appearance.density"),只存与默认值不同的项——
// 所以「已修改」标记与「恢复默认」就是「键在不在表里」。写盘是原子的(临时文件 + 改名),
// 每次修改都记一条 INFO 日志,排查「我明明改了设置」时直接看日志。
package prefs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

var log = diag.L("prefs")

// Store 是偏好存储。零值不可用,用 Open / Memory 创建。并发安全。
type Store struct {
	mu   sync.RWMutex
	path string // 空 = 只在内存(演示与测试)
	vals map[string]any
}

// Memory 创建只存内存的偏好(不读写磁盘)。
func Memory() *Store { return &Store{vals: map[string]any{}} }

// Open 读取 path 里的偏好;文件不存在算空,损坏的文件改名留档后从空开始(不丢用户数据也不阻塞启动)。
func Open(path string) *Store {
	s := &Store{path: path, vals: map[string]any{}}
	b, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Warn("读取设置失败", "path", path, "err", err)
		}
		return s
	}
	if err := json.Unmarshal(b, &s.vals); err != nil {
		bak := path + ".broken-" + time.Now().Format("20060102-150405")
		log.Error("设置文件损坏,已改名留档并使用默认值", "path", path, "bak", bak, "err", err)
		_ = os.Rename(path, bak)
		s.vals = map[string]any{}
	}
	log.Info("载入设置", "path", path, "项数", len(s.vals))
	return s
}

// Has 报告键是否被用户改过(与默认值不同)。
func (s *Store) Has(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.vals[key]
	return ok
}

// Int 读整数,没有则返回 def。JSON 里的数字是 float64,这里统一转回。
func (s *Store) Int(key string, def int) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	switch v := s.vals[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return def
}

// Bool 读布尔。
func (s *Store) Bool(key string, def bool) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.vals[key].(bool); ok {
		return v
	}
	return def
}

// Str 读字符串。
func (s *Store) Str(key, def string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.vals[key].(string); ok {
		return v
	}
	return def
}

// Set 写入一项并落盘。值等于默认值时调用 Reset 更干净,这里由调用方决定。
func (s *Store) Set(key string, v any) {
	s.mu.Lock()
	old, had := s.vals[key]
	if had && old == v {
		s.mu.Unlock()
		return
	}
	s.vals[key] = v
	s.mu.Unlock()
	log.Info("设置已修改", "key", key, "value", v)
	s.save()
}

// Reset 恢复某项的默认值(删掉键)。
func (s *Store) Reset(key string) {
	s.mu.Lock()
	_, had := s.vals[key]
	delete(s.vals, key)
	s.mu.Unlock()
	if had {
		log.Info("设置已恢复默认", "key", key)
		s.save()
	}
}

// ResetAll 清空全部偏好。
func (s *Store) ResetAll() {
	s.mu.Lock()
	n := len(s.vals)
	s.vals = map[string]any{}
	s.mu.Unlock()
	log.Warn("全部设置已恢复默认", "清除项数", n)
	s.save()
}

// Keys 返回已修改的全部键名(拷贝)。
func (s *Store) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.vals))
	for k := range s.vals {
		out = append(out, k)
	}
	return out
}

// ExportJSON 导出当前偏好为 JSON(调用方应剔除密钥相关键)。
func (s *Store) ExportJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make(map[string]any, len(s.vals))
	for k, v := range s.vals {
		cp[k] = v
	}
	return json.MarshalIndent(cp, "", "  ")
}

// ImportPatch 用 JSON 对象合并进当前偏好(不覆盖路径时仍落盘)。
func (s *Store) ImportPatch(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	s.mu.Lock()
	for k, v := range raw {
		s.vals[k] = v
	}
	s.mu.Unlock()
	s.save()
	return nil
}

// Len 是被改过的项数。
func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.vals)
}

// save 原子写盘:先写同目录临时文件再改名,中途崩溃不会留下半截文件。
func (s *Store) save() {
	if s.path == "" {
		return
	}
	defer diag.SpanSlow("prefs", "写设置", 30*time.Millisecond)()
	s.mu.RLock()
	b, err := json.MarshalIndent(s.vals, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		log.Error("序列化设置失败", "err", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		log.Error("创建设置目录失败", "err", err)
		return
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		log.Error("写设置失败", "path", tmp, "err", err)
		return
	}
	if err := os.Rename(tmp, s.path); err != nil {
		log.Error("替换设置文件失败", "err", err)
	}
}
