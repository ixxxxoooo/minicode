package claude

import "testing"

// 应用自己的一次性调用留下的旧会话不进历史列表;用户自己的提问不受影响。
func TestIsInternalPrompt(t *testing.T) {
	yes := []string{
		"你是 Git 提交说明助手。根据下面的改动写一条提交说明:",
		"  你是代码补全助手。只输出要接在光标后的补全文本(不含已有前缀)",
		"你是代码编辑助手。只输出要替换进编辑器的代码片段,不要解释",
	}
	no := []string{"", "检查 main.go 里有没有未使用的函数", "你是谁?", "帮我写一个 Git 提交说明"}
	for _, s := range yes {
		if !isInternalPrompt(s) {
			t.Errorf("应判为内部提示词:%q", s)
		}
	}
	for _, s := range no {
		if isInternalPrompt(s) {
			t.Errorf("不该判为内部提示词:%q", s)
		}
	}
}
