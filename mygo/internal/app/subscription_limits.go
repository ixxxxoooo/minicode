package app

import (
	"os"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/claudecfg"
)

// subscriptionLimitsEligible 当前会话是否可能从 Claude 官方账号读到订阅限额(get_usage)。
// 以实际在用的模型与通道为准:仅配置了第三方但未启用、或机器上残留的 AWS_PROFILE 等变量,仍算官方。
func (a *App) subscriptionLimitsEligible() bool {
	if a == nil {
		return false
	}
	if a.cuDemo() {
		return true
	}
	if thirdPartyMappingActive(a) {
		return false
	}
	if id := strings.TrimSpace(a.st.Model); id != "" && !officialModelID(id) {
		return false
	}
	settings := claudecfg.MergedClaudeEnv(strings.TrimSpace(a.proj))
	for _, src := range []map[string]string{settings, processCloudEnv()} {
		for _, k := range cloudChannelKeys {
			if envTruthy(src[k]) {
				return false
			}
		}
	}
	if thirdPartyAnthropicEnv(settings) || thirdPartyAnthropicEnv(processAnthropicEnv()) {
		return false
	}
	return true
}

var cloudChannelKeys = []string{
	"CLAUDE_CODE_USE_BEDROCK",
	"CLAUDE_CODE_USE_VERTEX",
	"CLAUDE_CODE_USE_FOUNDRY",
}

// thirdPartyMappingActive 映射已启用且密钥与 Base URL 就绪(与 claudecfg.ProviderEnv 一致)。
func thirdPartyMappingActive(a *App) bool {
	if a == nil {
		return false
	}
	p := a.pvLoad()
	return p.store.MappingStatus(p.keys) == ""
}

func officialModelID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return true
	}
	if strings.HasPrefix(id, "claude-") {
		return true
	}
	switch id {
	case "opus", "sonnet", "haiku", "default":
		return true
	}
	return false
}

func envTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func thirdPartyAnthropicEnv(m map[string]string) bool {
	if m == nil {
		return false
	}
	if u := strings.TrimSpace(m["ANTHROPIC_BASE_URL"]); u != "" && !claudecfg.OfficialAnthropicBaseURL(u) {
		return true
	}
	if strings.TrimSpace(m["ANTHROPIC_AUTH_TOKEN"]) != "" {
		return true
	}
	return false
}

func processCloudEnv() map[string]string {
	out := map[string]string{}
	for _, k := range cloudChannelKeys {
		if v := os.Getenv(k); v != "" {
			out[k] = v
		}
	}
	return out
}

func processAnthropicEnv() map[string]string {
	out := map[string]string{}
	for _, k := range []string{"ANTHROPIC_BASE_URL", "ANTHROPIC_AUTH_TOKEN"} {
		if v := os.Getenv(k); v != "" {
			out[k] = v
		}
	}
	return out
}
