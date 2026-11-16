package app

import (
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func TestPromptQueue(t *testing.T) {
	var q promptQueue
	if q.add("  \n") {
		t.Fatal("空白消息不应排队")
	}
	q.add("一")
	q.add("二")
	q.add("三")
	if s, _ := q.next(); s != "一" || q.len() != 2 {
		t.Fatalf("队首应先出:%q len=%d", s, q.len())
	}
	// 中断:暂停,不再自动发送
	q.turnEnded(false)
	if !q.paused {
		t.Fatal("中断后应暂停")
	}
	if _, ok := q.next(); ok {
		t.Fatal("暂停中不应自动取出")
	}
	// 手动取第 2 条仍可
	if s, ok := q.take(1); !ok || s != "三" {
		t.Fatalf("take(1) = %q %v", s, ok)
	}
	// 正常结束恢复
	q.turnEnded(true)
	if s, ok := q.next(); !ok || s != "二" {
		t.Fatalf("恢复后应取出二:%q %v", s, ok)
	}
	if q.paused || q.len() != 0 {
		t.Fatal("取空后应清除暂停")
	}
	q.add("抢发")
	if !q.stopAndSend(0) {
		t.Fatal("stopAndSend 应成功")
	}
	if q.preempt != "抢发" || q.len() != 0 {
		t.Fatalf("preempt=%q len=%d", q.preempt, q.len())
	}
	q.turnEnded(false)
	if q.paused {
		t.Fatal("没有排队时不需要暂停")
	}
}

// 引导:发出后先藏起会话流里的回显,其后出现新的非用户条目(且过了最短停留)才放出来;回合结束立刻放出。
func TestSteerPending(t *testing.T) {
	var q promptQueue
	t0 := time.Now()
	items := []core.Item{&core.User{ID: "u1", Text: "第一条"}, &core.Assistant{ID: "a1"}}
	q.addSteer("改一下", "改一下", len(items), 0, t0)
	// 后端回显了用户消息:绑定 id,仍在等待
	items = append(items, &core.User{ID: "u2", Text: "改一下"})
	q.syncSteers(items, true, t0.Add(10*time.Millisecond))
	if !q.held("u2") || len(q.steers) != 1 {
		t.Fatalf("应绑定并隐藏回显: %+v", q.steers)
	}
	// 其后出现新步骤,但没到最短停留:仍然等待
	items = append(items, &core.ToolCall{ID: "t1"})
	q.syncSteers(items, true, t0.Add(100*time.Millisecond))
	if len(q.steers) != 1 {
		t.Fatal("未到最短停留不应放出")
	}
	q.syncSteers(items, true, t0.Add(steerDwell+time.Millisecond))
	if len(q.steers) != 0 || q.held("u2") {
		t.Fatal("接收后应放进会话流")
	}
	// 回合结束:立刻放出
	q.addSteer("再改", "再改", len(items), 0, t0)
	q.syncSteers(items, false, t0)
	if len(q.steers) != 0 {
		t.Fatal("回合结束应放出")
	}
}

// 超过 steerMaxWait 仍不满足启发式时,一律从待发送区放出。
func TestSteerMaxWait(t *testing.T) {
	var q promptQueue
	t0 := time.Now()
	items := []core.Item{&core.User{ID: "u1", Text: "首条"}, &core.Assistant{ID: "a1"}}
	q.addSteer("卡住", "卡住", len(items), 0, t0)
	items = append(items, &core.User{ID: "u2", Text: "卡住"})
	q.syncSteers(items, true, t0.Add(steerDwell+time.Second))
	if len(q.steers) != 1 {
		t.Fatal("仅有回显、后面没有新步骤时不应放出")
	}
	q.syncSteers(items, true, t0.Add(steerMaxWait+time.Millisecond))
	if len(q.steers) != 0 || q.held("u2") {
		t.Fatal("超时后应放出")
	}
}
