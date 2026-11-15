package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/claudecfg"
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
