package app

import "testing"

// 崩溃记录类别要翻成中文,不认识的原样保留。
func TestCrashKindLabel(t *testing.T) {
	cases := map[string]string{
		"panic":         "程序崩溃",
		"hang":          "界面卡死",
		"unclean-exit":  "未正常退出",
		"abnormal-exit": "异常退出",
		"ips":           "系统崩溃报告",
		"":              "记录",
		"weird-kind":    "weird-kind",
	}
	for k, want := range cases {
		if got := crashKindLabel(k); got != want {
			t.Errorf("crashKindLabel(%q)=%q,想要 %q", k, got, want)
		}
	}
}

func TestFmtPerfDur(t *testing.T) {
	if got := fmtPerfDur(20290 * 1000); got != "20.29 ms" {
		t.Errorf("毫秒档: %q", got)
	}
	if got := fmtPerfDur(1269660 * 1000); got != "1.27 s" {
		t.Errorf("秒档: %q", got)
	}
}
