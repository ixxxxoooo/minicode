package app

import "testing"

// 设置里禁止了「绕过权限」:权限弹层里那一行写明原因,点它不发模式切换,而是去设置。
func TestBypassBlockedRowExplains(t *testing.T) {
	a, tt := quickApp(t)
	fb := &fakeBE{}
	a.Attach(fb)
	a.prefs.Set(pPermNoByp, true)
	a.togglePop("cfg")
	tt.Frame()
	if !tt.HasText("设置中已禁止") {
		t.Fatal("被禁止的行应写明「设置中已禁止」")
	}
	before := len(fb.cmds)
	if err := tt.Click("跳过权限确认"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(fb.cmds) != before {
		t.Fatal("被禁止时不应发出模式切换命令")
	}
	if a.popup == "cfg" {
		t.Fatal("点击后弹层应关闭并转到设置")
	}
}
