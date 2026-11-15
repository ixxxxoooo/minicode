package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// 一次性调用必须带 --no-session-persistence,否则每次生成提交说明都会在
// ~/.claude/projects 里多出一个会话,污染用户的历史列表(真实模式实测发现)。
func TestRunPromptModelNoSessionPersistence(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "claude")
	// 假 claude:把收到的参数原样回显到标准输出
	if err := os.WriteFile(fake, []byte("#!/bin/sh\necho \"$@\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MINI_CLAUDE_PATH", fake)
	out, err := RunPromptModel(dir, "你好", "plan", "haiku", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "--no-session-persistence") {
		t.Fatalf("参数里缺 --no-session-persistence:%q", out)
	}
}
