package core

import "testing"

// 第三方模型的 CLI 估算费用不展示;Claude 系列与未记录模型照常展示。
func TestCostShownOnlyForClaude(t *testing.T) {
	c := 0.17
	cases := map[string]bool{"deepseek-chat": false, "claude-sonnet-5-5": true, "": true, "<synthetic>": true, "my-opus-proxy": true}
	for model, want := range cases {
		ts := TurnStats{Input: 10, Output: 5, Model: model, CostUSD: &c}
		got := containsDollar(ts.ShortText())
		if got != want {
			t.Errorf("模型 %q:展示费用=%v,期望 %v(%s)", model, got, want, ts.ShortText())
		}
	}
}

func containsDollar(s string) bool {
	for _, r := range s {
		if r == '$' {
			return true
		}
	}
	return false
}
