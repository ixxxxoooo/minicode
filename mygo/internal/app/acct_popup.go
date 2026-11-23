package app

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ixxxxoooo/minicode/internal/claudecfg"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/diag"
)

var acctLog = diag.L("app.acct")

type acctState struct {
	loading      bool
	profileReady bool
	name         string
	subtitle     string
	initial      string
	monthCost    string // 空 = 暂无数据
	fiveLabel    string
	fivePct      int // -1 = 不显示
}

func (a *App) acctMenuData() acctState {
	if a.cuDemo() {
		d := demo.AcctMenuDemo()
		return acctState{
			profileReady: true,
			name:         d.Name,
			subtitle:     d.Subtitle,
			initial:      d.Initial,
			monthCost:    d.MonthCost,
			fiveLabel:    d.FiveLabel,
			fivePct:      d.FivePct,
		}
	}
	return a.acct
}

func (a *App) acctRefresh(force bool) {
	if a.cuDemo() {
		return
	}
	if a.acct.loading {
		return
	}
	if !force && a.acct.profileReady {
		return
	}
	a.acct.loading = true
	a.bg("acct-profile", func() func() {
		defer diag.SpanSlow("app", "读账号信息", 300*time.Millisecond)()
		p, err := claudecfg.ReadOAuthProfile()
		return func() {
			a.acct.loading = false
			a.acct.profileReady = true
			if err != nil {
				acctLog.Warn("读取 claude.json 失败", "err", err)
			}
			a.acctApplyProfile(p)
			a.usageRefreshLimits(false)
			if strings.HasPrefix(a.popup, "acct@") { // 弹层没开就不取用量(打开时会自己取)
				a.acctApplyUsage()
			}
		}
	})
}

func (a *App) acctApplyProfile(p claudecfg.OAuthProfile) {
	name := strings.TrimSpace(p.DisplayName)
	if name == "" {
		name = strings.TrimSpace(p.Email)
	}
	if at := strings.Index(name, "@"); at > 0 {
		name = name[:at]
	}
	if name == "" {
		a.acctApplyThirdPartyOrGuest()
		return
	}
	a.acct.name = name
	a.acct.initial = acctInitial(name)
	a.acct.subtitle = acctSubtitle(p)
}

func acctInitial(name string) string {
	r, _ := utf8.DecodeRuneInString(strings.TrimSpace(name))
	if r == utf8.RuneError {
		return "?"
	}
	s := string(r)
	if len(s) > 0 {
		return strings.ToUpper(s[:1])
	}
	return "?"
}

// acctApplyThirdPartyOrGuest 没有 OAuth 时:若映射到第三方服务商则显示其名称,否则未登录空态。
func (a *App) acctApplyThirdPartyOrGuest() {
	st, _ := claudecfg.LoadStore()
	if st.Mapping.Enabled {
		if pr := st.Get(st.Mapping.Provider); pr != nil {
			a.acct.name = pr.Name
			a.acct.subtitle = "使用第三方服务商 · 无 Claude 官方账户"
			a.acct.initial = acctInitial(pr.Name)
			return
		}
	}
	a.acct.name = "未登录"
	a.acct.subtitle = "未找到 OAuth 账户"
	a.acct.initial = "?"
}

func acctSubtitle(p claudecfg.OAuthProfile) string {
	t := strings.TrimSpace(p.AccountType)
	switch {
	case strings.Contains(t, "max"):
		return "Claude Max · 官方账号"
	case strings.Contains(t, "pro"):
		return "Claude Pro · 官方账号"
	case t != "":
		label := strings.ToUpper(t[:1]) + t[1:]
		return "Claude " + label + " · 官方账号"
	}
	if strings.TrimSpace(p.Email) != "" {
		return "Claude 官方账号"
	}
	return "未找到账户信息"
}

func (a *App) acctApplyUsage() {
	if a.cuDemo() {
		return
	}
	if a.usageSrc != nil {
		u := a.usageSrc(2)
		if u.Total > 0 {
			a.acct.monthCost = core.FormatMoney(u.Total)
		} else {
			a.acct.monthCost = ""
		}
	}
	u := a.st.Usage
	a.acct.fiveLabel = "5 小时"
	if u.FiveHourPct != nil {
		a.acct.fivePct = int(*u.FiveHourPct + 0.5)
	} else {
		a.acct.fivePct = -1
	}
}
