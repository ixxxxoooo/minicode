package app

import (
	"path/filepath"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/claudecfg"
	"github.com/ixxxxoooo/minicode/internal/projects"
)

func TestSubscriptionLimitsEligible(t *testing.T) {
	cases := []struct {
		name string
		fn   func(a *App)
		want bool
	}{
		{"演示", func(a *App) {}, true},
		{"官方默认", func(a *App) { realReg(a, t) }, true},
		{"第三方映射", func(a *App) {
			realReg(a, t)
			p := a.pvLoad()
			p.store = claudecfg.Store{
				Providers: []claudecfg.Provider{{ID: "ds", Name: "DS", BaseURL: "https://x", Models: []string{"m1"}}},
				Mapping:   claudecfg.Mapping{Enabled: true, Provider: "ds", Default: "m1"},
			}
		}, false},
		{"Bedrock", func(a *App) {
			realReg(a, t)
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

// realReg 换成落盘注册表,让应用离开演示模式(演示模式下限额一律视为可展示)。
func realReg(a *App, t *testing.T) {
	t.Helper()
	a.SetRegistry(projects.Open(filepath.Join(t.TempDir(), "ws.json")))
}
