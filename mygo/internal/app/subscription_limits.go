package app

import (
	"os"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/claudecfg"
	"github.com/ixxxxoooo/minicode/internal/core"
)

// subscriptionLimitsEligible 当前会话是否可能从 Claude 官方账号读到订阅限额(get_usage)。
// 第三方服务商映射、自定义 ANTHROPIC_BASE_URL、Bedrock / Vertex 等不计入官方。
func (a *App) subscriptionLimitsEligible() bool {
	if a == nil {
		return false
	}
	if a.cuDemo() {
		return true
	}
	if a.pvActiveModel() != "" {
		return false
	}
	for _, e := range a.sessionProviderEnv() {
		k := strings.ToUpper(strings.TrimSpace(e.Key))
		switch k {
		case "ANTHROPIC_BASE_URL":
			if v := strings.TrimSpace(e.Value); v != "" {
				return false
			}
		case "CLAUDE_CODE_USE_BEDROCK", "AWS_REGION", "AWS_PROFILE":
			if strings.TrimSpace(e.Value) != "" {
				return false
			}
		case "GOOGLE_CLOUD_PROJECT", "GOOGLE_CLOUD_LOCATION", "GOOGLE_APPLICATION_CREDENTIALS",
			"ANTHROPIC_VERTEX_PROJECT_ID", "CLOUD_ML_REGION":
			if strings.TrimSpace(e.Value) != "" {
				return false
			}
		}
	}
	return true
}

func (a *App) sessionProviderEnv() []core.EnvVar {
	if a == nil {
		return nil
	}
	var out []core.EnvVar
	for _, e := range claudecfg.ProviderEnv() {
		out = append(out, core.EnvVar{Key: e.Key, Value: e.Value})
	}
	out = append(out, a.RuntimeExtraEnv()...)
	for _, k := range []string{
		"CLAUDE_CODE_USE_BEDROCK", "AWS_REGION", "AWS_PROFILE",
		"GOOGLE_CLOUD_PROJECT", "GOOGLE_CLOUD_LOCATION", "ANTHROPIC_VERTEX_PROJECT_ID",
	} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			out = append(out, core.EnvVar{Key: k, Value: v})
		}
	}
	return out
}
