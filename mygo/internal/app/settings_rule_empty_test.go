package app

import "testing"

// 权限规则输入框不预填示例;空输入点「添加」只提示、不写入任何规则。
func TestRuleDraftNotPrefilled(t *testing.T) {
	a, tt := newTestApp(t)
	a.openSettings("perm")
	tt.Frame()
	if a.set.ruleDraft != "" {
		t.Fatalf("规则输入框不应预填:%q", a.set.ruleDraft)
	}
	a.submitRuleAdd()
	if a.toast == "" {
		t.Error("空输入应给出提示")
	}
	if a.set.ruleDraft != "" {
		t.Errorf("空输入后草稿应仍为空:%q", a.set.ruleDraft)
	}
}
