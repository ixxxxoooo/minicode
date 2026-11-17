package diag

import "testing"

func TestClassifyCmd(t *testing.T) {
	if classifyCmd("/usr/bin/claude mcp") != "Claude" {
		t.Fatal("claude")
	}
	if classifyCmd("gopls serve") != "语言服务" {
		t.Fatal("lsp")
	}
	if classifyCmd("-zsh") != "终端" {
		t.Fatal("shell")
	}
}
