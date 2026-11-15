package claude

import (
	"strings"
	"testing"
)

func TestForkOptionsArgs(t *testing.T) {
	o := Options{}
	ForkConfig{ParentSessionID: "p", NewSessionID: "n"}.Apply(&o)
	a := strings.Join(o.Args(), " ")
	if !strings.Contains(a, "--fork-session") || !strings.Contains(a, "--resume p") || !strings.Contains(a, "--session-id n") {
		t.Fatalf("分叉参数缺失:%s", a)
	}
}
