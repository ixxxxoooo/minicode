package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// 前台正在看会话时,权限请求只记进通知中心(已读),不弹浮层遮住权限卡。
func TestNeedsApprovalQuietInWork(t *testing.T) {
	a, _ := newTestApp(t)
	a.screen = ScreenWork
	a.noticeInit()
	before := len(a.nt.toasts)
	a.onNeedsApproval("权限请求", nil)
	if len(a.nt.toasts) != before {
		t.Fatalf("会话页不应弹浮层,toasts %d → %d", before, len(a.nt.toasts))
	}
	if len(a.nt.items) == 0 || !a.nt.items[0].read || a.nt.items[0].title != "权限请求" {
		t.Fatalf("通知中心应有一条已读的记录: %+v", a.nt.items)
	}
}

func TestPermToolTag(t *testing.T) {
	cases := map[string]string{"Edit": "编辑", "Bash": "命令", "Write": "写入", "mcp__x__y": "mcp__x__y"}
	for in, want := range cases {
		if got := permToolTag(&core.ToolCall{Name: in}); got != want {
			t.Errorf("%s → %s,期望 %s", in, got, want)
		}
	}
}
