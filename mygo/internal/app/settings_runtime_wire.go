package app

import (
	"strings"

	"github.com/ixxxxoooo/minicode/internal/claudecfg"
	"github.com/ixxxxoooo/minicode/internal/core"
)

const (
	pProxyMode  = "runtime.proxy"
	pProxyURL   = "runtime.proxyUrl"
	pNoProxy    = "runtime.noProxy"
	pRuntimeTO  = "runtime.timeout"
	pModelThink = "models.thinking"
	pModelCmp   = "models.compact"
	pModelCmpAt = "models.compactAt"
	pPasteFold  = "general.paste"
	pWinClose   = "general.close"
)

// defaultChatSys 与 settings_tabs.sysPromptArea 默认文案一致。
const defaultChatSys = "所有回答用中文,注释也用中文。\n提交信息使用 Conventional Commits。\n不要读取或输出任何密钥。"

// ChatSystemPrompt 追加系统提示词(空键时用默认示范文案)。
func (a *App) ChatSystemPrompt() string {
	if a == nil || a.prefs == nil {
		return defaultChatSys
	}
	return strings.TrimSpace(a.prefs.Str(pChatSys, defaultChatSys))
}

// RuntimeExtraEnv 返回应从偏好注入 claude 子进程的环境变量。
func (a *App) RuntimeExtraEnv() []core.EnvVar {
	if a == nil || a.prefs == nil {
		return nil
	}
	return claudecfg.RuntimeEnv(
		a.prefs.Int(pProxyMode, 0),
		a.prefs.Str(pProxyURL, ""),
		a.prefs.Str(pNoProxy, ""),
		a.prefs.Str(pRuntimeTO, ""),
	)
}

// ContextCompactThreshold 自动压缩触发比例(0–1),来自 models.compactAt,默认 0.85。
func (a *App) ContextCompactThreshold() float32 {
	if a == nil || a.prefs == nil {
		return 0.85
	}
	p := a.prefs.Int(pModelCmpAt, 85)
	if p < 50 {
		p = 50
	}
	if p > 95 {
		p = 95
	}
	return float32(p) / 100
}
