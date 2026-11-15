package app

import (
	"strings"
	"testing"
)

// 多仓库总览:全部同步逐个执行;有冲突的仓库被跳过并点名;失败的行留「重试」,重试成功后清掉;其它仓库不受影响。
func TestSyncAllReposStates(t *testing.T) {
	stressDemo(t)
	a, tt := newTestApp(t)
	a.ActivateWorkspace("ws-fullstack")
	a.OpenSCM()
	tt.Frame()
	if conf, _ := a.repoFlags(2); conf != 1 {
		t.Fatalf("docs-site 应有 1 个冲突文件:%d", conf)
	}
	if conf, _ := a.repoFlags(0); conf != 6 {
		t.Fatalf("router-demo 演示数据应有 6 个冲突文件:%d", conf)
	}
	a.syncAllRepos(false)
	toast := a.toast // 没有窗口时同步执行,汇总提示此刻就在
	tt.Frame()
	e1, e2 := a.repos[1], a.repos[2]
	if e1.opErr == "" || e1.retry != "同步" || e1.busy != "" {
		t.Fatalf("web-dashboard 第一次同步应失败并留重试:err=%q retry=%q busy=%q", e1.opErr, e1.retry, e1.busy)
	}
	if e2.opErr != "" || e2.busy != "" {
		t.Fatalf("docs-site 有冲突应被跳过而不是失败:%+v", e2)
	}
	if !strings.Contains(toast, "1 个失败") || !strings.Contains(toast, "跳过") || !strings.Contains(toast, "docs-site") || !strings.Contains(toast, "router-demo") {
		t.Fatalf("汇总提示不对:%q", toast)
	}
	if a.repos[0].opErr != "" {
		t.Fatal("router-demo 不应受 web-dashboard 失败的影响")
	}
	// 失败的行点「重试」:这次成功(演示的失败只发生一次),错误清掉
	a.repoRetry(1)
	tt.Frame()
	if e1.opErr != "" || e1.busy != "" {
		t.Fatalf("重试应成功:%+v", e1)
	}
	// 非仓库 / 越界不报错
	a.repoRetry(99)
	a.repoSync(99, true, true)
	// 总览行右键菜单 / 悬停渲染不崩
	if err := tt.RightClick("web-dashboard"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	menu := strings.Join(tt.Menu(), "|")
	for _, want := range []string{"设为活动仓库", "拉取", "推送", "同步", "切换分支…", "在图谱中查看", "储藏所有更改", "在访达中显示"} {
		if !strings.Contains(menu, want) {
			t.Fatalf("总览行右键菜单缺 %q:%s", want, menu)
		}
	}
}
