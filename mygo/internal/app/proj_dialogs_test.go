package app

import (
	"testing"
	"time"
)

func TestAddProjectDialogRenders(t *testing.T) {
	a, tt := newTestApp(t)
	a.OpenProjDialog("add")
	tt.Frame()
	tt.Advance(800 * time.Millisecond)
	t.Logf("kind=%q", a.pd.kind)
	if a.pd.kind != "addProj" {
		t.Fatalf("对话框不应自己关闭:kind=%q", a.pd.kind)
	}
	if _, ok := tt.Find("浏览…"); !ok {
		t.Fatal("应画出对话框(浏览按钮)")
	}
}

func TestConfirmDialogRenders(t *testing.T) {
	a, tt := newTestApp(t)
	a.OpenProjDialog("confirm")
	tt.Frame()
	time.Sleep(300 * time.Millisecond)
	tt.Frame()
	if _, ok := tt.Find("取消"); !ok {
		t.Fatal("确认框应画出")
	}
}
