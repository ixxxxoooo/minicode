package claudecfg

import "testing"

func TestOAuthProfileFields(t *testing.T) {
	oa := map[string]any{"displayName": "测试用户", "emailAddress": "u@example.com"}
	if strField(oa, "displayName") != "测试用户" || strField(oa, "emailAddress") != "u@example.com" {
		t.Fatal("字段解析")
	}
	if strField(oa, "missing") != "" {
		t.Fatal("缺失字段应为空")
	}
}
