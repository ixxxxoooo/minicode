package app

import (
	"strings"
	"testing"
)

// 连通性测试成功 / 失败应给出 toast(行内标签仍更新),且不泄露密钥。
func TestProviderConnectivityToast(t *testing.T) {
	t.Setenv("MINI_HOME", t.TempDir())
	a, tt := newTestApp(t)
	if !a.demoWs() {
		t.Skip("演示模式")
	}
	a.openSettings("models")
	tt.Frame()
	a.testProvider("deepseek")
	// shell 每帧会清空 toast,断言须在 Frame 之前
	if !strings.Contains(a.toast, "DeepSeek") || !strings.Contains(a.toast, "连接正常") || !strings.Contains(a.toast, "ms") {
		t.Fatalf("成功应有 toast:%q", a.toast)
	}
	if strings.Contains(a.toast, "demo-key") || strings.Contains(a.toast, "sk-") {
		t.Fatalf("toast 不应含密钥:%q", a.toast)
	}
	a.testProvider("gateway")
	if a.toast == "" || !strings.Contains(a.toast, "API Key") {
		t.Fatalf("失败应有可行动 toast:%q", a.toast)
	}
	_ = tt
}
