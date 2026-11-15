package claudecfg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

var log = diag.L("claudecfg")

// readJSON 读 JSON 对象;不存在返回空 map。
func readJSON(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	var v map[string]any
	if err := json.Unmarshal(b, &v); err != nil {
		bak := path + ".broken-" + time.Now().Format("20060102-150405")
		log.Error("JSON 损坏,已改名留档", "path", path, "bak", bak, "err", err)
		_ = os.Rename(path, bak)
		return map[string]any{}, nil
	}
	if v == nil {
		v = map[string]any{}
	}
	return v, nil
}

// writeJSON 原子写盘并备份上一份。
func writeJSON(path string, v map[string]any) error {
	defer diag.SpanSlow("claudecfg", "写配置", 30*time.Millisecond)()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		bak := path + ".mini-backup"
		_ = os.Remove(bak)
		_ = os.Rename(path, bak)
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func homeClaudeJSON() string {
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".claude.json")
}

func userSettings() string {
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".claude", "settings.json")
}

func projectKey(path string) string {
	path = filepath.Clean(path)
	if path != "/" && strings.HasSuffix(path, string(os.PathSeparator)) {
		path = path[:len(path)-1]
	}
	return path
}
