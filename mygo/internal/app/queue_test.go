package app

import "testing"

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
