package app

import (
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/automation"
)

// 没有任何自动化时(真实模式首次使用),点「新建自动化」要能打开对话框。
func TestAutomationNewFromEmpty(t *testing.T) {
	a, tt := newTestApp(t)
	a.SetAutomations(automation.NewStore(nil))
	a.SetMode(ModeIDE)
	a.screen = ScreenAutomation
	tt.Frame()
	if !tt.HasText("还没有自动化") {
		t.Fatalf("应显示空态:%v", tt.Texts())
	}
	if err := tt.Click("新建自动化"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(250 * time.Millisecond)
	tt.Frame()
	tt.Frame()
	if !a.au.newDlg {
		t.Fatal("点击后应打开新建对话框")
	}
	if !tt.HasText("创建") && !tt.HasText("取消") {
		t.Fatalf("对话框内容没画出来:%v", tt.Texts())
	}
}

// 新建对话框:错误按字段定位,填全后创建成功并关闭。
func TestAutomationNewValidate(t *testing.T) {
	a, tt := newTestApp(t)
	a.SetAutomations(automation.NewStore(nil))
	a.screen = ScreenAutomation
	tt.Frame()
	a.openAutoNewDlg()
	a.submitAutoNew()
	if a.au.newErrAt != "prompt" || a.au.newErr == "" {
		t.Fatalf("空提示词的错误应落在提示词:at=%q err=%q", a.au.newErrAt, a.au.newErr)
	}
	a.au.newPrompt, a.au.newTime = "检查依赖", "9点"
	a.submitAutoNew()
	if a.au.newErrAt != "time" {
		t.Fatalf("坏时间的错误应落在时间:%q", a.au.newErrAt)
	}
	a.au.newTime = "09:30"
	a.submitAutoNew()
	if a.au.newDlg || len(a.auto.List()) != 1 {
		t.Fatalf("应创建成功并关闭:dlg=%v n=%d", a.au.newDlg, len(a.auto.List()))
	}
	a.openAutoNewDlg()
	if a.au.newErr != "" {
		t.Fatal("重新打开应清掉上次的错误")
	}
}
