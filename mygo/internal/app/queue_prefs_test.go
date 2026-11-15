package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/prefs"
)

func TestQueueDraftFor(t *testing.T) {
	a := New(Options{Prefs: prefs.Memory()})
	raw := "你好 @src/main.go\n\n![图](a.png)"
	a.prefs.Set(pQueueAtt, false)
	got := a.queueDraftFor(raw)
	if got != "你好" {
		t.Fatalf("剥离附件后=%q", got)
	}
	a.prefs.Set(pQueueAtt, true)
	if a.queueDraftFor(raw) != raw {
		t.Fatal("开启时应原样保留")
	}
}

func TestQTurnEndedNoPause(t *testing.T) {
	a := New(Options{Prefs: prefs.Memory()})
	a.q.add("排队一")
	a.prefs.Set("chat.queuePause", false)
	a.qTurnEnded(false)
	if a.q.paused {
		t.Fatal("queuePause 关时不应暂停")
	}
}
