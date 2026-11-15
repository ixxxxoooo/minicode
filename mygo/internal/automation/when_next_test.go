package automation

import (
	"testing"
	"time"
)

func TestWhenNext(t *testing.T) {
	now := time.Date(2026, 11, 16, 3, 30, 0, 0, time.UTC) // 周一
	at := func(d, h int) time.Time { return time.Date(2026, 11, d, h, 0, 0, 0, time.UTC) }
	cases := []struct {
		t    time.Time
		want string
	}{
		{at(16, 9), "今天 09:00"},
		{at(17, 9), "明天 09:00"},
		{at(20, 18), "周五 18:00"},
		{at(23, 9), "11 月 23 日 09:00"},
	}
	for _, c := range cases {
		if got := WhenNext(now, c.t); got != c.want {
			t.Errorf("%v → %q,想要 %q", c.t, got, c.want)
		}
	}
}
