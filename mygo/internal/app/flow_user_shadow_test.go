package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func TestUserMsgCardBuildsWithShadow(t *testing.T) {
	a, tt := newTestApp(t)
	a.st.Items = append(a.st.Items, &core.User{ID: "u-shadow-test", Text: "重启"})
	tt.Frame()
	if !tt.HasText("重启") {
		t.Fatal("应渲染用户卡")
	}
}
