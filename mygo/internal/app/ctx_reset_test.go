package app

import (
	"testing"
	"time"
)

func TestResetIn(t *testing.T) {
	for _, c := range []struct {
		d    time.Duration
		want string
	}{
		{-time.Minute, "已重置"},
		{30 * time.Second, "不到 1 分钟后重置"},
		{25 * time.Minute, "25 分钟后重置"},
		{2*time.Hour + 10*time.Minute, "2 小时 10 分后重置"},
		{72 * time.Hour, "3 天后重置"},
	} {
		if got := resetIn(c.d); got != c.want {
			t.Fatalf("resetIn(%v)=%q,期望 %q", c.d, got, c.want)
		}
	}
}
