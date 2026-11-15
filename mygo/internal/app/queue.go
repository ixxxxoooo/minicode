package app

// 排队与引导(对标 Cursor):回合进行中发新消息默认排队,本轮正常结束后自动逐条发送;
// ⌘Enter 立即「引导」(塞进当前回合)。回合被中断 / 出错时队列暂停,等用户选择继续或撤回。

type queued struct {
	ID   int
	Text string
}

// promptQueue 是排队的消息:纯逻辑,不依赖界面。
type promptQueue struct {
	items  []queued
	paused bool
	seq    int
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

func (q *promptQueue) clear() { q.items, q.paused = nil, false }

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
