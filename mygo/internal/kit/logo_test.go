package kit

import "testing"

func TestBrandOf(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{[]string{"deepseek-chat"}, "deepseek"},
		{[]string{"DeepSeek-R1"}, "deepseek"},
		{[]string{"glm-4.6"}, "zhipu"},
		{[]string{"kimi-k2-turbo-preview"}, "kimi"},
		{[]string{"moonshot-v1-8k"}, "kimi"},
		{[]string{"qwen3-coder-plus"}, "qwen"},
		{[]string{"MiniMax-M2"}, "minimax"},
		{[]string{"claude-sonnet-5-5"}, "claude"},
		{[]string{"gpt-4o"}, "openai"},
		{[]string{"o3-mini"}, "openai"},
		{[]string{"gemini-2.5-pro"}, "gemini"},
		{[]string{"llama-3.1-70b"}, "meta"},
		{[]string{"step-2-16k"}, "stepfun"},
		{[]string{"yi-large"}, "yi"},
		// 服务商:名称优先,其次域名
		{[]string{"智谱 GLM", "open.bigmodel.cn"}, "zhipu"},
		{[]string{"自建网关", "api.deepseek.com"}, "deepseek"},
		{[]string{"Anthropic API", "api.anthropic.com"}, "claude"},
		// 认不出的不乱配
		{[]string{"custom-model-a"}, ""},
		{[]string{"企业内网网关", "llm.corp.example.com"}, ""},
		{[]string{""}, ""},
	}
	for _, c := range cases {
		if got := BrandOf(c.in...); got != c.want {
			t.Errorf("BrandOf(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// 规则里引用的品牌都必须有图形,且每个图形都能被解析成 SVG。
func TestLogosComplete(t *testing.T) {
	for _, r := range logoRules {
		if _, ok := logoBodies[r.Brand]; !ok {
			t.Errorf("品牌 %s 没有图形", r.Brand)
		}
		if logoSVG(r.Brand) == nil {
			t.Errorf("品牌 %s 解析失败", r.Brand)
		}
	}
	if logoSVG("不存在") != nil {
		t.Error("未知品牌应返回 nil(调用方画占位图标)")
	}
}
