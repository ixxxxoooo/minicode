package claudecfg

import (
	"strings"
)

// OAuthProfile 是从 ~/.claude.json 的 oauthAccount 读出的只读身份字段(不含 token / apiKey)。
type OAuthProfile struct {
	DisplayName string
	Email       string
	AccountType string // 如 max / pro;没有则为空
}

// ReadOAuthProfile 读取用户 OAuth 账户展示信息;文件不存在或没有 oauthAccount 时返回零值。
func ReadOAuthProfile() (OAuthProfile, error) {
	m, err := readJSON(homeClaudeJSON())
	if err != nil {
		return OAuthProfile{}, err
	}
	oa, _ := m["oauthAccount"].(map[string]any)
	if oa == nil {
		return OAuthProfile{}, nil
	}
	p := OAuthProfile{
		DisplayName: strField(oa, "displayName", "name"),
		Email:       strField(oa, "emailAddress", "email"),
		AccountType: strField(oa, "accountType", "plan", "subscriptionType"),
	}
	if p.AccountType != "" {
		p.AccountType = strings.ToLower(p.AccountType)
	}
	return p, nil
}

func strField(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok {
			v = strings.TrimSpace(v)
			if v != "" {
				return v
			}
		}
	}
	return ""
}
