package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/claudecfg"
	"github.com/ixxxxoooo/minicode/internal/projects"
)

func TestAcctMenuThirdPartyEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("MINI_HOME", home)
	path := filepath.Join(home, "providers.json")
	os.WriteFile(path, []byte(`{"providers":[{"id":"deepseek","name":"DeepSeek","baseUrl":"https://api.deepseek.com/anthropic","models":["deepseek-chat"]}],"mapping":{"enabled":true,"provider":"deepseek","default":"deepseek-chat"}}`), 0o600)

	a, _ := newTestApp(t)
	a.acctApplyThirdPartyOrGuest()
	if a.acct.name != "DeepSeek" || !strings.Contains(a.acct.subtitle, "第三方") {
		t.Fatalf("应显示第三方空态:%q %q", a.acct.name, a.acct.subtitle)
	}
}

func TestAcctMenuDataLoading(t *testing.T) {
	a, _ := newTestApp(t)
	// 演示模式返回演示账号数据,不反映 loading;换成落盘注册表走真实路径
	a.SetRegistry(projects.Open(filepath.Join(t.TempDir(), "ws.json")))
	a.demoExtra = false
	a.acct.loading = true
	d := a.acctMenuData()
	if !d.loading {
		t.Fatal("应报告加载中")
	}
}

func TestAcctApplyProfileOAuth(t *testing.T) {
	a, _ := newTestApp(t)
	a.acctApplyProfile(claudecfg.OAuthProfile{DisplayName: "Lin", Email: "lin@test.com", AccountType: "pro"})
	if a.acct.name != "Lin" || !strings.Contains(a.acct.subtitle, "Pro") {
		t.Fatalf("官方账号:%q %q", a.acct.name, a.acct.subtitle)
	}
}
