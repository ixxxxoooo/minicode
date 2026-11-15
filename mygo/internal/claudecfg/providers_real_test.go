package claudecfg

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// 真实服务商联调:读本机 ~/.minicode 里已配好的 deepseek 与密钥,请求真实接口。
// 默认跳过;运行:MINI_REAL_PROVIDER=1 go test ./internal/claudecfg -run Real -v
// 密钥只从本机磁盘读取,不进代码、不进日志、不进断言文案。
func realDeepseek(t *testing.T) (Store, Provider, map[string]string) {
	t.Helper()
	if os.Getenv("MINI_REAL_PROVIDER") != "1" {
		t.Skip("设置 MINI_REAL_PROVIDER=1 才跑真实服务商联调")
	}
	s, err := LoadStore()
	if err != nil {
		t.Fatal(err)
	}
	p := s.Get("deepseek")
	if p == nil {
		t.Skip("本机没有配置 deepseek 服务商")
	}
	keys, _ := LoadKeys()
	if strings.TrimSpace(keys["deepseek"]) == "" {
		t.Skip("本机没有 deepseek 的 API Key")
	}
	return s, *p, keys
}

func TestRealFetchModels(t *testing.T) {
	_, p, keys := realDeepseek(t)
	ids, err := FetchModels(p, keys["deepseek"])
	if err != nil {
		t.Fatalf("获取模型列表失败:%v", err)
	}
	t.Logf("DeepSeek 返回 %d 个模型:%v", len(ids), ids)
	if len(ids) == 0 {
		t.Fatal("模型列表为空")
	}
	// 错误的 Key 应当得到 401 一类的带原因错误,而不是成功
	if _, err := FetchModels(p, "invalid-key-for-test"); err == nil {
		t.Error("错误的 Key 不应获取成功")
	} else {
		t.Logf("错误 Key 的提示:%v", err)
	}
}

func TestRealProviderConnectivity(t *testing.T) {
	_, p, _ := realDeepseek(t)
	ok, ms := TestProvider(p.BaseURL)
	t.Logf("连通性:%s", FormatTestResult(ok, ms))
	if !ok {
		t.Fatal("DeepSeek 地址不可达")
	}
}

// 端到端:按映射注入的环境变量启动真实 claude,问一句话,确认经 DeepSeek 返回。
func TestRealClaudeViaProvider(t *testing.T) {
	s, _, keys := realDeepseek(t)
	if !s.Mapping.Enabled {
		t.Skip("本机映射未启用")
	}
	path, err := exec.LookPath("claude")
	if err != nil {
		t.Skip("没有找到 claude")
	}
	env := os.Environ()
	for _, e := range s.Env(keys) {
		env = append(env, e.Key+"="+e.Value)
	}
	cmd := exec.Command(path, "-p", "只回复两个字:你好", "--output-format", "text")
	cmd.Env = env
	cmd.Dir = t.TempDir()
	done := make(chan struct{})
	var out []byte
	go func() { out, err = cmd.CombinedOutput(); close(done) }()
	select {
	case <-done:
	case <-time.After(90 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("claude 90 秒内没有返回")
	}
	got := strings.TrimSpace(string(out))
	// 输出里不能带出密钥
	if k := keys["deepseek"]; k != "" && strings.Contains(got, k) {
		t.Fatal("输出里出现了密钥")
	}
	t.Logf("claude 经 DeepSeek 的回复:%q (err=%v)", got, err)
	if err != nil || got == "" {
		t.Fatalf("claude 调用失败:%v", err)
	}
}
