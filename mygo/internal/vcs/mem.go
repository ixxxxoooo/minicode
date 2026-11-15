package vcs

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Mem 是内存里的假仓库(演示 / 测试用)。
type Mem struct {
	mu      sync.Mutex
	cur     string
	br      []Branch
	changes []Change
	log     []Commit
	files   map[string][]Change // 提交 → 涉及的文件
	head    map[string]string   // 路径 → HEAD 里的内容
	msgs    []string            // 提交说明(演示:提交后把暂存区清掉并加一条)
	ahead   int
	behind  int
	index   map[string]string // 路径 → 暂存区里的内容(没有记录 = 与 HEAD 相同)
	stash   []Stash
	blame   map[string][]BlameLine       // 演示:按路径灌入追溯行
	fileLog map[string][]Commit          // 演示:按路径覆盖 FileLog
	fail    map[string]string            // 演示:操作名(Pull / Push)→ 失败原因,用来演示「某个仓库失败、其余照常」
	once    bool                         // 失败只发生一次(第二次「重试」成功)
	delay   time.Duration                // 演示:拉取 / 推送耗时,界面才看得到转圈
	rebase  bool                         // 演示:变基冲突中
	stashDt map[int]map[string][2]string // 演示:储藏条目→路径→[旧,新] diff 内容
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

// ───────── 演示数据 ─────────

// SetState 灌入演示用的改动、提交、各提交文件与 HEAD 内容。
func (m *Mem) SetState(changes []Change, log []Commit, files map[string][]Change, head map[string]string, ahead int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.changes, m.log, m.files, m.head, m.ahead = changes, log, files, head, ahead
}

// SetAheadBehind 演示:领先 / 落后提交数。
func (m *Mem) SetAheadBehind(ahead, behind int) {
	m.mu.Lock()
	m.ahead, m.behind = ahead, behind
	m.mu.Unlock()
}

// SetBranches 演示:替换分支列表。
func (m *Mem) SetBranches(br []Branch) {
	m.mu.Lock()
	m.br = append([]Branch(nil), br...)
	m.mu.Unlock()
}

// SetCurrent 演示:当前分支或游离 HEAD(短哈希)。
func (m *Mem) SetCurrent(name string) {
	m.mu.Lock()
	m.cur = name
	m.mu.Unlock()
}

// SetIndex 演示:暂存区内容;未列出的路径 Index 仍回落到 HEAD。
func (m *Mem) SetIndex(idx map[string]string) {
	m.mu.Lock()
	if m.index == nil {
		m.index = map[string]string{}
	}
	for k, v := range idx {
		m.index[k] = v
	}
	m.mu.Unlock()
}

func (m *Mem) Status() ([]Change, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Change(nil), m.changes...), nil
}

func (m *Mem) move(paths []string, staged bool) {
	for _, p := range paths {
		for i := range m.changes {
			c := &m.changes[i]
			if c.Path == p && c.Staged != staged {
				if staged && c.Status == 'U' {
					c.Status = 'A'
				}
				c.Staged = staged
				break
			}
		}
	}
}

func (m *Mem) Stage(paths ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.move(paths, true)
	return nil
}
func (m *Mem) Unstage(paths ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.move(paths, false)
	return nil
}

func (m *Mem) Discard(p string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, c := range m.changes {
		if c.Path == p && !c.Staged {
			m.changes = append(m.changes[:i:i], m.changes[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("没有可放弃的改动:%s", p)
}

func (m *Mem) Commit(msg string, amend bool) error {
	if err := m.slow("Commit"); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if msg == "" {
		return fmt.Errorf("提交说明不能为空")
	}
	var rest, staged []Change
	for _, c := range m.changes {
		if c.Staged {
			staged = append(staged, c)
		} else {
			rest = append(rest, c)
		}
	}
	if len(staged) == 0 && !amend {
		return fmt.Errorf("没有暂存的更改")
	}
	m.changes = rest
	hash := fmt.Sprintf("d%06x", len(m.log)*7919+len(msg))
	m.log = append([]Commit{{Hash: hash, Subject: msg, Author: "你", When: "刚刚", Parents: headParent(m.log), Refs: []string{"HEAD", m.cur}}}, stripHead(m.log, m.cur)...)
	if m.files == nil {
		m.files = map[string][]Change{}
	}
	m.files[hash] = staged
	m.ahead++
	return nil
}

func headParent(l []Commit) []string {
	if len(l) == 0 {
		return nil
	}
	return []string{l[0].Hash}
}

// stripHead 新提交成为 HEAD 后,旧的 HEAD / 当前分支引用要从原提交上拿掉。
func stripHead(l []Commit, branch string) []Commit {
	out := append([]Commit(nil), l...)
	for i := range out {
		var refs []string
		for _, r := range out[i].Refs {
			if r != "HEAD" && r != branch {
				refs = append(refs, r)
			}
		}
		out[i].Refs = refs
	}
	return out
}

func (m *Mem) Log(n int) ([]Commit, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if n > len(m.log) {
		n = len(m.log)
	}
	return append([]Commit(nil), m.log[:n]...), nil
}

func (m *Mem) CommitFiles(hash string) ([]Change, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Change(nil), m.files[hash]...), nil
}

func (m *Mem) Show(rev, p string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.head[p]
	return s, ok
}

func (m *Mem) AheadBehind() (int, int) { m.mu.Lock(); defer m.mu.Unlock(); return m.ahead, m.behind }

// SetFail 演示 / 测试:让某个操作(Pull / Push)失败,reason 为空则恢复正常;once 为真时只失败第一次。
func (m *Mem) SetFail(op, reason string, once bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail == nil {
		m.fail = map[string]string{}
	}
	m.once = once
	if reason == "" {
		delete(m.fail, op)
	} else {
		m.fail[op] = reason
	}
}

// SetDelay 演示:让拉取 / 推送各耗时 d(界面上的转圈才看得到)。
func (m *Mem) SetDelay(d time.Duration) { m.mu.Lock(); m.delay = d; m.mu.Unlock() }

func (m *Mem) slow(op string) error {
	m.mu.Lock()
	d, reason := m.delay, m.fail[op]
	if reason != "" && m.once {
		delete(m.fail, op)
	}
	m.mu.Unlock()
	if d > 0 {
		time.Sleep(d)
	}
	if reason != "" {
		return errors.New(reason)
	}
	return nil
}

func (m *Mem) Pull() error {
	if err := m.slow("Pull"); err != nil {
		return err
	}
	m.mu.Lock()
	m.behind = 0
	m.mu.Unlock()
	return nil
}

func (m *Mem) PullRebase() error {
	if err := m.slow("PullRebase"); err != nil {
		if strings.Contains(err.Error(), "冲突") || strings.Contains(strings.ToLower(err.Error()), "conflict") {
			m.mu.Lock()
			m.rebase = true
			m.changes = append(m.changes, Change{Path: "src/router.rs", Status: StatusConflict, Add: 2, Del: 2})
			m.mu.Unlock()
		}
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.behind = 0
	m.rebase = false
	return nil
}

func (m *Mem) PullMerge() error {
	if err := m.slow("PullMerge"); err != nil {
		if strings.Contains(err.Error(), "冲突") || strings.Contains(strings.ToLower(err.Error()), "conflict") {
			m.mu.Lock()
			m.changes = append(m.changes, Change{Path: "src/router.rs", Status: StatusConflict, Add: 2, Del: 2})
			m.mu.Unlock()
		}
		return err
	}
	m.mu.Lock()
	m.behind = 0
	m.mu.Unlock()
	return nil
}

func (m *Mem) RebaseAbort() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.rebase {
		return fmt.Errorf("当前没有进行中的变基")
	}
	m.rebase = false
	for i := range m.changes {
		if m.changes[i].Status == StatusConflict {
			m.changes = append(m.changes[:i], m.changes[i+1:]...)
			break
		}
	}
	return nil
}

func (m *Mem) Push() error {
	if err := m.slow("Push"); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ahead = 0
	return nil
}

func (m *Mem) PushForce() error {
	if err := m.slow("Push"); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ahead = 0
	return nil
}

// SetBranch 演示用:把当前分支设成 name(不存在就加一个),并设定落后数。
func (m *Mem) SetBranch(name string, behind int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	found := false
	for _, b := range m.br {
		if !b.Remote && b.Name == name {
			found = true
		}
	}
	if !found {
		m.br = append(m.br, Branch{Name: name, When: "刚刚"})
	}
	m.cur, m.behind = name, behind
}
