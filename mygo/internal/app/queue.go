package app

import (
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// 排队与引导(对标 Cursor):回合进行中发新消息默认排队,本轮正常结束后自动逐条发送;
// ⌘Enter 立即「引导」(塞进当前回合)。回合被中断 / 出错时队列暂停,等用户选择继续或撤回。

type queued struct {
	ID   int
	Text string
}

// promptQueue 是排队的消息:纯逻辑,不依赖界面。
type promptQueue struct {
	items   []queued
	paused  bool
	seq     int
	preempt string // 停止并发送:中断结束后优先发出的那条
	steers  []*steerPend
}

// steerPend 是已经发给 Claude、但还没被它接收的「引导」消息:先显示在输入框上方的待发送区,
// 被接收后才在会话流里露面(这之前会话流里隐藏后端回显的那条用户消息)。
type steerPend struct {
	Text string    // 正文(与后端回显的 User.Text 对得上);图片消息可能为空
	Show string    // 待发送区里显示的文字
	from int       // 发出时的会话条目数,只在它之后找回显
	uid  string    // 后端回显的用户消息 id(找到前为空)
	at   time.Time // 发出时刻(最短停留,避免一闪而过)
}

// steerDwell 引导行至少停留这么久再放进会话流。
const steerDwell = 700 * time.Millisecond

// addSteer 登记一条刚发出的引导。
func (q *promptQueue) addSteer(text, show string, from int, now time.Time) {
	q.steers = append(q.steers, &steerPend{Text: text, Show: show, from: from, at: now})
}

// held 这条会话流用户消息是否正是一条尚未被接收的引导(界面据此隐藏它)。
func (q *promptQueue) held(uid string) bool {
	for _, s := range q.steers {
		if s.uid != "" && s.uid == uid {
			return true
		}
	}
	return false
}

// syncSteers 对照会话流更新引导:找到回显的用户消息就绑定 id;其后出现新的非用户条目(Claude 已接着往下走)
// 或回合结束,即视为已接收,放进会话流。后端没有显式确认,这是启发式。
func (q *promptQueue) syncSteers(items []core.Item, running bool, now time.Time) {
	if len(q.steers) == 0 {
		return
	}
	claimed := map[string]bool{}
	for _, s := range q.steers {
		if s.uid != "" {
			claimed[s.uid] = true
		}
	}
	keep := q.steers[:0]
	for _, s := range q.steers {
		at := -1
		if s.uid == "" {
			for k := s.from; k < len(items); k++ {
				if u, ok := items[k].(*core.User); ok && u.Text == s.Text && !claimed[u.ID] {
					s.uid, at = u.ID, k
					claimed[u.ID] = true
					break
				}
			}
		} else {
			for k, it := range items {
				if it.ItemID() == s.uid {
					at = k
					break
				}
			}
		}
		done := !running
		if running && at >= 0 && now.Sub(s.at) >= steerDwell {
			for k := at + 1; k < len(items); k++ {
				if _, isUser := items[k].(*core.User); !isUser {
					done = true
					break
				}
			}
		}
		if !done {
			keep = append(keep, s)
		}
	}
	q.steers = keep
}

// add 排入一条;空白消息忽略。
func (q *promptQueue) add(text string) bool {
	if len(trim(text)) == 0 {
		return false
	}
	q.seq++
	q.items = append(q.items, queued{ID: q.seq, Text: text})
	return true
}

func (q *promptQueue) len() int { return len(q.items) }

// take 取出第 i 条(退回输入框编辑 / 引导 / 立即发送)。
func (q *promptQueue) take(i int) (string, bool) {
	if i < 0 || i >= len(q.items) {
		return "", false
	}
	t := q.items[i].Text
	q.items = append(q.items[:i:i], q.items[i+1:]...)
	if len(q.items) == 0 {
		q.paused = false
	}
	return t, true
}

// next 取出队首;暂停中不取。
func (q *promptQueue) next() (string, bool) {
	if q.paused {
		return "", false
	}
	return q.take(0)
}

// turnEnded 回合结束:正常结束保持 / 恢复自动发送,否则暂停(有排队时)。
func (q *promptQueue) turnEnded(normal bool) {
	if normal {
		q.paused = false
	} else if len(q.items) > 0 {
		q.paused = true
	}
}

func (q *promptQueue) clear() { q.items, q.paused, q.preempt = nil, false, "" }

// reset 切会话时清空一切(含引导)。
func (q *promptQueue) reset() { q.clear(); q.steers = nil }

// stopAndSend 取出第 i 条并在中断结束后发出。
func (q *promptQueue) stopAndSend(i int) bool {
	s, ok := q.take(i)
	if !ok {
		return false
	}
	q.preempt = s
	q.paused = false
	return true
}

func trim(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\n' || s[i] == '\t' || s[i] == '\r') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\n' || s[j-1] == '\t' || s[j-1] == '\r') {
		j--
	}
	return s[i:j]
}
