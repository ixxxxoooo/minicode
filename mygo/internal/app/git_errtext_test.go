package app

import (
	"strings"
	"testing"
)

func TestFriendlyGitErr(t *testing.T) {
	cases := []struct{ raw, want string }{
		{"fatal: No configured push destination.\nEither specify the URL from the command-line\n  git remote add <name> <url>", "还没有配置远程仓库"},
		{"fatal: The current branch main has no upstream branch.", "上游分支"},
		{"fatal: Authentication failed for 'https://x/y.git'", "认证失败"},
		{"fatal: unable to access 'https://x/': Could not resolve host: x", "连不上远程仓库"},
		{"! [rejected] main -> main (non-fast-forward)", "推送被拒绝"},
		{"*** Please tell me who you are.", "Git 用户"},
		{"fatal: Unable to create '/r/.git/index.lock': File exists.", "index.lock"},
	}
	for _, c := range cases {
		if got := friendlyGitErr(c.raw); !strings.Contains(got, c.want) {
			t.Errorf("%q → %q,应含 %q", c.raw, got, c.want)
		}
	}
	// 认不出的:只留第一行、去 fatal 前缀、不含换行
	got := friendlyGitErr("fatal: something odd\nsecond line\nthird")
	if strings.Contains(got, "\n") || strings.Contains(got, "fatal") || !strings.Contains(got, "something odd") {
		t.Errorf("兜底格式不对:%q", got)
	}
	if long := friendlyGitErr("error: " + strings.Repeat("长", 300)); len([]rune(long)) > 160 {
		t.Errorf("未截断:%d", len([]rune(long)))
	}
	if friendlyGitErr("") == "" {
		t.Error("空输入应有兜底文案")
	}
}
