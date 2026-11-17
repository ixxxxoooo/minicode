package claude

import (
	"testing"
	"time"
)

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

// 同一工作目录分散在多个 claude 项目目录里(从别处 --resume 过来)时合成一个项目,同 id 会话只留最新一份。
func TestMergeByCwd(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	in := []ProjectMeta{
		{Cwd: "/a/MiniCode", Sessions: []SessionMeta{{ID: "s1", Modified: t0}, {ID: "s2", Modified: t0}}},
		{Cwd: "/a/MiniCode/", Sessions: []SessionMeta{{ID: "s1", Modified: t0.Add(time.Hour)}}},
		{Cwd: "/a/Other", Sessions: []SessionMeta{{ID: "s3", Modified: t0}}},
	}
	out := mergeByCwd(in)
	if len(out) != 2 {
		t.Fatalf("应合成 2 个项目,实际 %d", len(out))
	}
	if n := len(out[0].Sessions); n != 2 || out[0].Sessions[0].ID != "s1" || !out[0].Sessions[0].Modified.Equal(t0.Add(time.Hour)) {
		t.Errorf("同 id 会话应去重并留最新:%+v", out[0].Sessions)
	}
}
