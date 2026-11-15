// 演示数据索引(设置)
//
//   - Providers / Env / Rules / Trust:设置 → 模型与服务商、权限与信任、环境变量。
//   - LSPRows:设置 → 语言服务器(演示行覆盖真实探测)。
//   - ShortcutPrefs:设置 → 快捷键(含冲突与自定义组合)。
//   - ClaudeSummary:设置 → 高级 → Claude 诊断摘要(钩子 / 插件 / MCP 汇总)。
package demo

import (
	"github.com/ixxxxoooo/minicode/internal/claudecfg"
)

// DemoProviders 第三方服务商各种连通状态(密钥在界面只展示 sk-••••)。
func DemoProviders() []claudecfg.Provider {
	return []claudecfg.Provider{
		{ID: "anthropic", Name: "Anthropic 官方(默认)", BaseURL: "https://api.anthropic.com", Active: true, Status: claudecfg.FormatTestResult(true, 142)},
		{ID: "deepseek", Name: "DeepSeek", BaseURL: "https://api.deepseek.com/anthropic", Active: false, Status: claudecfg.FormatTestResult(true, 89)},
		{ID: "openai-compat", Name: "OpenAI 兼容", BaseURL: "https://api.openai.com/v1", Active: false, Status: claudecfg.ProvFail},
		{ID: "corp", Name: "企业内网网关", BaseURL: "https://llm.corp.example.com", Active: false, Status: claudecfg.ProvTestIng},
		{ID: "disabled", Name: "已禁用服务商", BaseURL: "https://legacy.example.com", Active: false, Status: "已禁用"},
		{ID: "custom", Name: "自建网关", BaseURL: "https://llm.example.com", Status: claudecfg.ProvUnverified},
	}
}

// ProviderKeyMasks 只用于展示,不存真实密钥。
func ProviderKeyMasks() map[string]string {
	return map[string]string{
		"anthropic": "sk-••••••••abcd", "deepseek": "sk-••••", "openai-compat": "sk-••••",
		"corp": "sk-••••••", "custom": "",
	}
}

// SettingDemoRules 权限规则:长规则、重复规则、空组以外俱全。
func SettingDemoRules() claudecfg.PermRules {
	return claudecfg.PermRules{
		Allow: []string{
			"Bash(git status)", "Bash(cargo test:*)", "Read(src/**)",
			"WebFetch(domain:docs.rs)", "WebFetch(domain:github.com)",
			"Bash(git log --oneline -n 50)",
		},
		Ask: []string{
			"Bash(git push:*)", "Bash(npm publish:*)",
			"Write(./.github/workflows/**)",
		},
		Deny: []string{
			"Read(.env)", "Read(**/.env*)", "Bash(rm -rf:*)",
			"Bash(curl * | sh)", "Write(/etc/**)",
			"Bash(git push origin main --force)",
			"Read(/Users/**/.ssh/**)",
		},
	}
}

// DemoEnv 环境变量列表(值一律遮盖)。
func DemoEnv() []claudecfg.EnvRow {
	return []claudecfg.EnvRow{
		{Name: "ANTHROPIC_API_KEY", Scope: "用户", Mask: "sk-••••••••"},
		{Name: "ANTHROPIC_BASE_URL", Scope: "用户", Mask: "https://••••••••"},
		{Name: "HTTPS_PROXY", Scope: "用户", Mask: "http://••••:8080"},
		{Name: "ENABLE_TOOL_SEARCH", Scope: "项目", Mask: "true"},
		{Name: "API_TIMEOUT_MS", Scope: "本地", Mask: "120000"},
		{Name: "NODE_OPTIONS", Scope: "项目", Mask: "••••••••"},
	}
}

// DemoTrust 项目信任行(路径用演示目录)。
func DemoTrust() []claudecfg.TrustEntry {
	return []claudecfg.TrustEntry{
		{Path: "/demo/router-demo", Trusted: true, HookOK: true, McpOK: true},
		{Path: "/demo/web-dashboard", Trusted: true, HookOK: true, McpOK: false},
		{Path: "/demo/docs-site", Trusted: false, HookOK: false, McpOK: false},
		{Path: "/demo/plain-notes", Trusted: true, HookOK: false, McpOK: true},
		{Path: "/demo/mega-sessions", Trusted: false, HookOK: false, McpOK: false},
		{Path: "/Volumes/OldDisk/Projects/removed-repo", Trusted: false, HookOK: false, McpOK: false},
	}
}

// LSPDemoRow 是设置页语言服务器一行的演示数据。
type LSPDemoRow struct {
	Name   string
	Status string
	OK     bool
}

// LSPDemoRows 已安装 / 未安装 / 过旧 / 启动失败 / 索引中。
func LSPDemoRows() []LSPDemoRow {
	return []LSPDemoRow{
		{Name: "rust-analyzer", Status: "已就绪 · v2025.12.1", OK: true},
		{Name: "gopls", Status: "已就绪 · v0.18.0", OK: true},
		{Name: "Pyright", Status: "未安装 · pip install pyright", OK: false},
		{Name: "typescript-language-server", Status: "版本过旧 · 需要 ≥ 4.0", OK: false},
		{Name: "clangd", Status: "启动失败 · 上次退出码 1", OK: false},
		{Name: "SourceKit-LSP", Status: "索引中 · 42%", OK: true},
	}
}

// ShortcutDemoPrefs 自定义快捷键与冲突示例(pref 键 → 组合文案)。
func ShortcutDemoPrefs() map[string]string {
	return map[string]string{
		"shortcuts.palette": "⌘ E", // 与切换 IDE 冲突
		"shortcuts.search":  "⌘ K", // 占位 command 面板默认
		"shortcuts.save":    "⌘ ⇧ S",
	}
}

// ClaudeDiagLines 高级页 Claude 诊断摘要(钩子 / 插件 / MCP)。
func ClaudeDiagLines() []string {
	return []string{
		"钩子:用户层 2 条 · 项目层 1 条 · 本地 0 条",
		"插件:已启用 3 / 市场 12 · filesystem 1.2.0 · github 0.9.1",
		"MCP:已连接 4 · 失败 1 · 按需加载工具 6",
		"市场连接正常 · 上次检查 3 分钟前",
	}
}
