package app

import "testing"

func TestSubscriptionLimitsEligible(t *testing.T) {
	cases := []struct {
		name string
		fn   func(a *App)
		want bool
	}{
		{"演示", func(a *App) { a.runDemo = true }, true},
		{"官方默认", func(a *App) {}, true},
		{"第三方映射", func(a *App) {
			a.pvLoad().store.Activate("deepseek", "deepseek-reasoner")
		}, false},
		{"Bedrock", func(a *App) {
			t.Setenv("CLAUDE_CODE_USE_BEDROCK", "1")
		}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := New(Options{})
			c.fn(a)
			got := a.subscriptionLimitsEligible()
			if got != c.want {
				t.Fatalf("eligible=%v 要 %v", got, c.want)
			}
		})
	}
}
