package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func realSkip(t *testing.T) {
	t.Helper()
	if os.Getenv("MINI_REAL_CLAUDE") != "1" {
		t.Skip("设置 MINI_REAL_CLAUDE=1 才跑真实 claude")
	}
	if _, ok := FindClaude(""); !ok {
		t.Skip("本机没有 claude")
	}
}

// realRigWithCleanup 用真实 ~/.claude/projects 落盘,便于分叉 / 恢复;测试结束删除会话目录。
func realRigWithCleanup(t *testing.T, cfg Config) (*rig, string) {
	t.Helper()
	realSkip(t)
	if cfg.Cwd == "" {
		cfg.Cwd = t.TempDir()
	}
	r := startRealRig(t, cfg)
	t.Cleanup(func() {
		cleanClaudeSession(t, r, cfg.Cwd)
	})
	return r, cfg.Cwd
}

// waitTimeout 在持锁下轮询 cond;cond 必须在已持有 r.mu 时调用,不得再锁 r.mu。
func (r *rig) waitTimeout(timeout time.Duration, what string, cond func() bool) {
	r.t.Helper()
	deadline := time.Now().Add(timeout)
	timer := time.AfterFunc(timeout, func() { r.mu.Lock(); r.cond.Broadcast(); r.mu.Unlock() })
	defer timer.Stop()
	r.mu.Lock()
	defer r.mu.Unlock()
	for !cond() {
		if time.Now().After(deadline) {
			r.t.Fatalf("等待超时:%s\n事件数:%d", what, len(r.evs))
		}
		r.cond.Wait()
	}
}

func (r *rig) turnsEndedLocked() int {
	n := 0
	for _, e := range r.evs {
		if _, ok := e.(core.EvTurnEnded); ok {
			n++
		}
	}
	return n
}

func (r *rig) hasEventLocked(pred func(core.Event) bool) bool {
	for _, e := range r.evs {
		if pred(e) {
			return true
		}
	}
	return false
}

func (r *rig) usageCountLocked() int {
	n := 0
	for _, e := range r.evs {
		if _, ok := e.(core.EvUsage); ok {
			n++
		}
	}
	return n
}

func skipIfLimitReply(t *testing.T, r *rig) {
	t.Helper()
	if strings.Contains(strings.ToLower(replyOf(r)), "limit") {
		t.Skip("账号触及限额,跳过")
	}
	for _, e := range r.evs {
		if n, ok := e.(core.EvNotice); ok && strings.Contains(strings.ToLower(n.Text), "limit") {
			t.Skip("账号触及限额,跳过")
		}
	}
}

func (r *rig) lastUsage() (core.Usage, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := len(r.evs) - 1; i >= 0; i-- {
		if u, ok := r.evs[i].(core.EvUsage); ok {
			return u.Usage, true
		}
	}
	return r.st.Usage, true
}

func (r *rig) userUUIDs() []core.EvUserUUID {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []core.EvUserUUID
	for _, e := range r.evs {
		if u, ok := e.(core.EvUserUUID); ok {
			out = append(out, u)
		}
	}
	return out
}

func (r *rig) hasEvent(pred func(core.Event) bool) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.evs {
		if pred(e) {
			return true
		}
	}
	return false
}

func (r *rig) runningTasks() []*core.BackgroundTask {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*core.BackgroundTask(nil), r.st.Tasks...)
}

func sessionJSONL(projects, cwd, sess string) string {
	dir := filepath.Join(projects, EncodeProjectDir(cwd))
	return filepath.Join(dir, sess+".jsonl")
}
