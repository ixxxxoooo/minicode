package app

import (
	"os"
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
		{"配置了但未启用第三方", func(a *App) {
			realReg(a, t)
			p := a.pvLoad()
			p.store = claudecfg.Store{
				Providers: []claudecfg.Provider{{ID: "ds", Name: "DS", BaseURL: "https://x", Models: []string{"m1"}}},
				Mapping:   claudecfg.Mapping{Enabled: false, Provider: "ds", Default: "m1"},
			}
		}, true},
		{"启用第三方", func(a *App) {
			realReg(a, t)
			p := a.pvLoad()
			p.store = claudecfg.Store{
				Providers: []claudecfg.Provider{{ID: "ds", Name: "DS", BaseURL: "https://x", Models: []string{"m1"}}},
				Mapping:   claudecfg.Mapping{Enabled: true, Provider: "ds", Default: "m1"},
			}
			p.keys = map[string]string{"ds": "k"}
		}, false},
		{"仅有 AWS_PROFILE", func(a *App) {
			realReg(a, t)
			t.Setenv("AWS_PROFILE", "default")
			t.Setenv("AWS_REGION", "us-east-1")
			t.Setenv("GOOGLE_CLOUD_PROJECT", "demo")
		}, true},
		{"USE_BEDROCK=1", func(a *App) {
			realReg(a, t)
			t.Setenv("CLAUDE_CODE_USE_BEDROCK", "1")
		}, false},
		{"settings 里有 BASE_URL", func(a *App) {
			realReg(a, t)
			home := t.TempDir()
			dir := filepath.Join(home, ".claude")
			_ = os.MkdirAll(dir, 0o700)
			body := `{"env":{"ANTHROPIC_BASE_URL":"https://api.deepseek.com/anthropic"}}`
			_ = os.WriteFile(filepath.Join(dir, "settings.json"), []byte(body), 0o600)
			os.Setenv("HOME", home)
		}, false},
		{"模型 claude-* 映射启用", func(a *App) {
			realReg(a, t)
			p := a.pvLoad()
			p.store = claudecfg.Store{
				Providers: []claudecfg.Provider{{ID: "ds", Name: "DS", BaseURL: "https://x", Models: []string{"claude-sonnet-5-5"}}},
				Mapping:   claudecfg.Mapping{Enabled: true, Provider: "ds", Default: "claude-sonnet-5-5"},
			}
			p.keys = map[string]string{"ds": "k"}
			a.st.Model = "claude-sonnet-5-5"
		}, false},
		{"会话模型非官方 id", func(a *App) {
			realReg(a, t)
			a.st.Model = "deepseek-chat"
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

// TestSubscriptionLimitsEligibleRealMachine 只读检查本机真实 ~/.claude(需 MINI_PROBE_HOME=真实 HOME)。
func TestSubscriptionLimitsEligibleRealMachine(t *testing.T) {
	realHome := os.Getenv("MINI_PROBE_HOME")
	if realHome == "" {
		t.Skip("设置 MINI_PROBE_HOME 指向真实用户目录可跑一次只读核对")
	}
	old := os.Getenv("HOME")
	os.Setenv("HOME", realHome)
	t.Cleanup(func() { _ = os.Setenv("HOME", old) })
	a := New(Options{})
	realReg(a, t)
	got := a.subscriptionLimitsEligible()
	t.Logf("真实配置下 subscriptionLimitsEligible=%v (无密钥输出)", got)
}

// realReg 换成落盘注册表,让应用离开演示模式(演示模式下限额一律视为可展示)。
func realReg(a *App, t *testing.T) {
	t.Helper()
	a.SetRegistry(projects.Open(filepath.Join(t.TempDir(), "ws.json")))
}
