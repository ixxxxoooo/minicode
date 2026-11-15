package app

import "testing"

// 没写语言的围栏代码块不能让会话流 panic(曾因 strings.Fields(info)[0] 越界崩溃)。
func TestIsMermaidInfoEmpty(t *testing.T) {
	for info, want := range map[string]bool{"": false, "  ": false, "go": false, "mermaid": true, " Mermaid title": true} {
		if got := isMermaidInfo(info); got != want {
			t.Fatalf("isMermaidInfo(%q)=%v,期望 %v", info, got, want)
		}
	}
}
