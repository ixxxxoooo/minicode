package app

import (
	"path/filepath"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/projects"
)

// 回归:账号按钮每帧都会构建(连同弹层),原先每帧都读一次用量数据源;会话的每个用量事件也会读。
// 读数据源会被当成「有人在看用量」,后台用量扫描因此永远不空闲(开发版实测:没打开用量页,仍每分钟扫一次)。
// 现在弹层关着时一次都不读;打开后照常取到本月费用。
func TestAcctPopupReadsUsageOnlyWhenOpen(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // 账号资料读 ~/.claude.json:不碰本机真实文件
	a, tt := newTestApp(t)
	a.SetRegistry(projects.Open(filepath.Join(t.TempDir(), "projects.json"))) // 落盘存储 = 非演示
	a.demoExtra = false
	if a.cuDemo() {
		t.Fatal("测试前提:应处于非演示状态,否则不会去读用量数据源")
	}
	calls := 0
	a.SetUsageSource(func(int) core.UsageReport { calls++; return core.UsageReport{Total: 1.23} })

	a.popup = ""
	for i := 0; i < 5; i++ {
		tt.Frame()
	}
	a.onUsageEvent(core.Usage{}) // 会话的用量事件
	if calls != 0 {
		t.Fatalf("账号弹层关着时不应读用量数据源,读了 %d 次", calls)
	}

	// 打开弹层(侧栏齿轮 / 标题栏头像,哪个在当前布局里都行):应取到本月费用
	for _, key := range []string{"acct@gear", "acct@avatar"} {
		a.popup = key
		tt.Frame()
		if calls > 0 {
			break
		}
	}
	if calls == 0 {
		t.Fatal("打开账号弹层后应读用量数据源")
	}
	if want := core.FormatMoney(1.23); a.acct.monthCost != want {
		t.Fatalf("打开后应显示本月费用 %q,得到 %q", want, a.acct.monthCost)
	}
}
