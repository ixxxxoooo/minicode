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
