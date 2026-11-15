package claudecfg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRuntimeEnvProxy(t *testing.T) {
	env := RuntimeEnv(2, "http://127.0.0.1:7890", "localhost", "120")
	keys := map[string]string{}
	for _, e := range env {
		keys[e.Key] = e.Value
	}
	if keys["HTTP_PROXY"] != "http://127.0.0.1:7890" || keys["NO_PROXY"] != "localhost" {
		t.Fatalf("代理变量不对:%v", keys)
	}
	if keys["API_TIMEOUT_MS"] != "120000" {
		t.Fatalf("超时应为 120000 ms,得 %q", keys["API_TIMEOUT_MS"])
	}
	direct := RuntimeEnv(1, "", "", "")
	if len(direct) < 4 {
		t.Fatalf("直连应清空代理变量,得 %d 项", len(direct))
	}
}

func TestSyncSessionPrefs(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	_ = os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	old := os.Getenv("HOME")
	t.Setenv("HOME", home)
	defer func() { _ = os.Setenv("HOME", old) }()

	on := true
	deny := true
	err := SyncSessionPrefs(SessionPrefs{
		Layer:                 LayerUser,
		AlwaysThinking:        &on,
		DefaultMode:           "plan",
		DisableBypass:         &deny,
		AdditionalDirectories: []string{"/tmp/extra"},
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := readJSON(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if root["alwaysThinkingEnabled"] != true {
		t.Fatalf("alwaysThinkingEnabled 未写入:%v", root["alwaysThinkingEnabled"])
	}
	perm, _ := root["permissions"].(map[string]any)
	if perm == nil || perm["defaultMode"] != "plan" {
		t.Fatalf("defaultMode 不对:%v", perm)
	}
	if perm["disableBypassPermissionsMode"] != "disable" {
		t.Fatalf("disableBypass 不对:%v", perm["disableBypassPermissionsMode"])
	}
}
