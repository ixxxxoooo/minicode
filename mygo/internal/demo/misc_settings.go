// 演示数据索引(设置)
//
//   - Providers / Env / Rules / Trust:设置 → 模型与服务商、权限与信任、环境变量。
//   - LSPRows:设置 → 语言服务器(演示行覆盖真实探测)。
//   - ShortcutPrefs:设置 → 快捷键(含冲突与自定义组合)。
//   - ClaudeSummary:设置 → 高级 → Claude 诊断摘要(钩子 / 插件 / MCP 汇总)。
package demo

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/claudecfg"
)

// DemoProviderStore 演示用的服务商与模型映射:覆盖「正常 / 清单为空 / 缺密钥 / 失败 / 不支持列表 / 测试中」。
// 只在内存里用,不落盘。
func DemoProviderStore() claudecfg.Store {
	ds := []string{"deepseek-chat", "deepseek-reasoner", "deepseek-v3"}
	kimi := []string{"kimi-k2-turbo-preview", "kimi-k2-0905-preview", "kimi-k2-thinking", "moonshot-v1-8k", "moonshot-v1-32k", "moonshot-v1-128k",
		"moonshot-v1-auto", "kimi-latest", "kimi-thinking-preview", "moonshot-v1-8k-vision-preview", "moonshot-v1-32k-vision-preview", "moonshot-v1-128k-vision-preview"}
	return claudecfg.Store{
		Providers: []claudecfg.Provider{
			{ID: "deepseek", Name: "DeepSeek", BaseURL: "https://api.deepseek.com/anthropic", ModelsURL: "https://api.deepseek.com/models",
				Models: []string{"deepseek-chat", "deepseek-reasoner"}, Available: append([]string(nil), ds...),
				FetchedAt: time.Now().Unix(), Status: claudecfg.ProvOK},
			{ID: "glm-empty", Name: "智谱 GLM", BaseURL: "https://open.bigmodel.cn/api/anthropic", Status: claudecfg.ProvUnverified},
			{ID: "legacy", Name: "Legacy 网关", BaseURL: "https://api.legacy.local/anthropic", Models: []string{"custom-model-a"}, Manual: []string{"custom-model-a"},
				Status: claudecfg.ProvUnverified},
			{ID: "gateway", Name: "自建网关", BaseURL: "https://llm.example.com/v1", Status: claudecfg.ProvTestNone},
			// 以下为额外演示态:多项模型 / 连通成功 / 失败 / 测试中
			{ID: "kimi", Name: "Kimi", BaseURL: "https://api.moonshot.cn/anthropic", Models: []string{"kimi-k2-turbo-preview"}, Available: kimi,
				FetchedAt: time.Now().Add(-3 * time.Hour).Unix(), Status: claudecfg.FormatTestResult(true, 142)},
			{ID: "corp", Name: "企业内网网关", BaseURL: "https://llm.corp.example.com", Auth: claudecfg.AuthXAPIKey, Models: []string{"corp-large"}, Manual: []string{"corp-large"},
				Status: claudecfg.ProvTestIng},
			{ID: "bad", Name: "失效示例", BaseURL: "https://llm.example.com/v1", Status: claudecfg.ProvFail},
		},
		Mapping: claudecfg.Mapping{
			Enabled: true, Provider: "deepseek",
			Default: "deepseek-chat", Opus: "deepseek-reasoner", Sonnet: "deepseek-chat", Haiku: "deepseek-chat",
		},
	}
}

// ProviderKeyMasks 只用于展示,不存真实密钥。
func ProviderKeyMasks() map[string]string {
	return map[string]string{
		"deepseek": "sk-••••••••abcd", "glm-empty": "••••", "kimi": "sk-••••", "legacy": "sk-••••", "corp": "sk-••••••", "bad": "sk-••••", "gateway": "",
	}
}

// DemoProviderKeys 演示用的「密钥」:只有有没有的区别,值是占位,不会发往任何地方。
func DemoProviderKeys() map[string]string {
	m := map[string]string{}
	for id, mask := range ProviderKeyMasks() {
		if mask != "" {
			m[id] = "demo-key"
		}
	}
	return m
}

// DemoFetchModels 模拟「获取模型列表」:地址含 example.com 回 401,含 legacy 视为没有列表接口,其余按服务商返回一组模型。
func DemoFetchModels(p claudecfg.Provider) ([]string, error) {
	switch {
	case strings.Contains(p.BaseURL, "example.com"):
		return nil, errors.New("HTTP 401:invalid api key")
	case strings.Contains(p.BaseURL, "legacy"):
		return nil, claudecfg.ErrModelsUnsupported
	case strings.Contains(p.BaseURL, "deepseek"):
		return []string{"deepseek-chat", "deepseek-reasoner", "deepseek-v3"}, nil
	case strings.Contains(p.BaseURL, "bigmodel"):
		return []string{"glm-4.6", "glm-4.5", "glm-4.5-air", "glm-4.5-flash", "glm-4-plus", "glm-4-long"}, nil
	case strings.Contains(p.BaseURL, "moonshot"):
		return []string{"kimi-k2-turbo-preview", "kimi-k2-0905-preview", "kimi-k2-thinking", "moonshot-v1-8k", "moonshot-v1-32k", "moonshot-v1-128k"}, nil
	}
	ids := make([]string, 0, 28)
	for i := 1; i <= 28; i++ {
		ids = append(ids, "model-"+strconv.Itoa(i))
	}
	return ids, nil
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

// SettingTrustList 设置页项目信任:演示数据,可用 MINI_SET_TRUST=empty|fail 覆盖。
func SettingTrustList() ([]claudecfg.TrustEntry, error) {
	switch os.Getenv("MINI_SET_TRUST") {
	case "empty":
		return nil, nil
	case "fail":
		return nil, errors.New("演示:无法读取信任列表")
	default:
		return DemoTrust(), nil
	}
}

// TrustWriteFail 演示写失败(MINI_SET_TRUST=writefile)。
func TrustWriteFail() bool { return os.Getenv("MINI_SET_TRUST") == "writefile" }

// LSPDemoRow 是设置页语言服务器一行的演示数据。
type LSPDemoRow struct {
	ID      string
	Name    string
	Lang    string
	Exts    string
	Status  string
	Path    string
	Install string
	OK      bool
	Enabled bool
}

// LSPDemoRows 已安装 / 未安装 / 过旧 / 启动失败 / 索引中。
func LSPDemoRows() []LSPDemoRow {
	return []LSPDemoRow{
		{ID: "rust", Name: "rust-analyzer", Lang: "Rust", Exts: ".rs", Status: "已安装", Path: "/opt/homebrew/bin/rust-analyzer", OK: true, Enabled: true},
		{ID: "go", Name: "gopls", Lang: "Go", Exts: ".go", Status: "已安装", Path: "~/go/bin/gopls", OK: true, Enabled: true},
		{ID: "python", Name: "Pyright", Lang: "Python", Exts: ".py, .pyi", Status: "未安装", Install: "npm install -g pyright", OK: false, Enabled: true},
		{ID: "ts", Name: "typescript-language-server", Lang: "TypeScript / JavaScript", Exts: ".ts, .tsx, .js", Status: "未安装", Install: "npm install -g typescript-language-server", OK: false, Enabled: false},
		{ID: "c", Name: "clangd", Lang: "C / C++", Exts: ".c, .cpp, …", Status: "已安装", Path: "/usr/bin/clangd", OK: true, Enabled: true},
		{ID: "swift", Name: "SourceKit-LSP", Lang: "Swift", Exts: ".swift", Status: "已安装", Path: "/usr/bin/sourcekit-lsp", OK: true, Enabled: true},
		// 下面三行演示「托管 Node 与语言包」的各态:下载中(设置页另有任务状态)、已装(托管)、已装但有新版本
		{ID: "php", Name: "Intelephense", Lang: "PHP", Exts: ".php, .phtml", Status: "未安装", Install: "npm install -g intelephense", OK: false, Enabled: true},
		{ID: "bash", Name: "bash-language-server", Lang: "Shell", Exts: ".sh, .bash, .zsh", Status: "已安装", Path: "~/.minicode/lsp/bash/node_modules/.bin/bash-language-server", OK: true, Enabled: true},
		{ID: "yaml", Name: "yaml-language-server", Lang: "YAML", Exts: ".yaml, .yml", Status: "可更新", Path: "~/.minicode/lsp/yaml/node_modules/.bin/yaml-language-server", OK: true, Enabled: true},
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
