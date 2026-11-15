package app

import "testing"

// 点导航里的「搜索」分段应让输入框自动聚焦,点别的分段不应触发。
func TestFocusSearchOnEnter(t *testing.T) {
	a := &App{}
	a.focusSearchOnEnter("files")
	if a.sr.focusNext {
		t.Fatal("切到文件不应请求搜索框焦点")
	}
	a.focusSearchOnEnter("search")
	if !a.sr.focusNext {
		t.Fatal("切到搜索应请求搜索框焦点")
	}
}
