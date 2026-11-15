package app

import (
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/automation"
)

// 运行记录弹窗停在「实时」键上、运行刚结束(实时记录已收掉)时,要自动改看最新一条记录,
// 不能落到空记录显示「运行失败,未产生会话 / 没有输出」(真实模式实测发现)。
func TestAutoRunViewFollowsLatestAfterFinish(t *testing.T) {
	a, tt := newTestApp(t)
	st := automation.NewStore(nil)
	a.SetAutomations(st)
	it := st.Add("demo")
	it.Name = "巡检"
	if err := st.Update(it); err != nil {
		t.Fatal(err)
	}
	st.Record(it.ID, automation.Run{At: time.Now().Add(-time.Hour), OK: false, Err: "旧失败"})
	st.Record(it.ID, automation.Run{At: time.Now(), OK: true, Output: "没有未使用的函数", SessionID: "sess-9"})
	a.screen = ScreenAutomation
	a.au.sel = it.ID
	a.au.runView = liveKey(it.ID)
	time.Sleep(250 * time.Millisecond)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("没有未使用的函数") {
		t.Fatalf("应显示最新一条记录的输出:%v", tt.Texts())
	}
	if tt.HasText("运行失败,未产生会话") || tt.HasText("没有输出") {
		t.Fatalf("不应落到空记录:%v", tt.Texts())
	}
	if !tt.HasText("查看会话") {
		t.Fatalf("最新记录有会话,应给「查看会话」:%v", tt.Texts())
	}
}
