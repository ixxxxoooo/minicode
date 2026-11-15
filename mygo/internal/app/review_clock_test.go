package app

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// 真实应用没有注入时钟(a.clock == nil):审查页页脚不能直接调 a.clock(),
// 否则点「审查」会让整个应用空指针闪退(真实模式实测发现)。
func TestReviewFooterWithoutClock(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	if a.clock != nil {
		t.Fatal("前提:未注入时钟")
	}
	if got := a.reviewFooterText(nil); !strings.Contains(got, "已查看 0 / 0") {
		t.Fatalf("页脚文案不对:%q", got)
	}
	tt := ui.NewTester(a.View, 1280, 800)
	tt.Frame()
	a.OpenReview()
	for i := 0; i < 3; i++ {
		tt.Frame() // 不应 panic
	}
}
