package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/ixxxxoooo/minicode/internal/diag"
)

// 设置备份(.mcsettings):偏好键值的 JSON,导出与导入都剔除密钥类键。
// 双击 .mcsettings 与「设置 → 高级 → 导入…」走同一条路径。

// settingsBackupExt 是设置备份文件的扩展名。
const settingsBackupExt = ".mcsettings"

// maxSettingsFile 设置备份大小上限(偏好只有几十 KB)。
const maxSettingsFile = 2 << 20

// isSecretPrefKey 判断偏好键是不是密钥一类:这些键不进备份,也不从备份里读。
func isSecretPrefKey(k string) bool {
	l := strings.ToLower(k)
	for _, w := range []string{"secret", "token", "password", "passwd", "credential", "apikey", "api_key", "api-key"} {
		if strings.Contains(l, w) {
			return true
		}
	}
	return strings.HasSuffix(l, ".key") || strings.HasSuffix(l, "_key")
}

// stripSecretPrefs 剔除密钥类键,返回剔除后的副本与被剔除的个数。
func stripSecretPrefs(raw map[string]json.RawMessage) (map[string]json.RawMessage, int) {
	out := make(map[string]json.RawMessage, len(raw))
	dropped := 0
	for k, v := range raw {
		if isSecretPrefKey(k) {
			dropped++
			continue
		}
		out[k] = v
	}
	return out, dropped
}

// exportSettingsJSON 导出设置备份:写到「下载」并在访达里显示。
func (a *App) exportSettingsJSON() {
	if a.prefs == nil {
		return
	}
	data, err := a.prefs.ExportJSON()
	if err != nil {
		a.toast = "导出失败:" + err.Error()
		setLog.Warn("导出设置失败", "err", err)
		return
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		a.toast = "导出失败:" + err.Error()
		return
	}
	clean, dropped := stripSecretPrefs(raw)
	out, err := json.MarshalIndent(clean, "", "  ")
	if err != nil {
		a.toast = "导出失败:" + err.Error()
		return
	}
	dest := uniqueFile(downloadsDir(), "MiniCode 设置备份", settingsBackupExt)
	if err := writeFileAtomic(dest, out); err != nil {
		a.toast = "导出失败:" + err.Error()
		setLog.Warn("写入设置备份失败", "path", dest, "err", err)
		return
	}
	a.toast = "已导出到「下载」:" + filepath.Base(dest) + "(不含密钥)"
	setLog.Info("导出设置", "path", dest, "keys", len(clean), "droppedSecrets", dropped)
	a.revealInFinder(dest)
}

// askImportSettings 「导入…」:选一个 .mcsettings(或 .json)再确认。
func (a *App) askImportSettings() {
	if a.win == nil || a.demoWs() {
		a.toast = "演示模式没有文件选择器"
		return
	}
	a.bg("pick-settings", func() func() {
		defer diag.SpanSlow("settings", "选择设置备份", time.Minute)()
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title: "选择设置备份", ButtonLabel: "选择", Parent: a.win,
			Filters: []mygo.FileFilter{{Name: "设置备份", Extensions: []string{"mcsettings", "json"}}},
		})
		return func() {
			if err != nil {
				setLog.Warn("设置备份选择器失败", "err", err)
				return
			}
			if len(paths) > 0 {
				a.importSettingsFile(paths[0])
			}
		}
	})
}

// importSettingsFile 读备份、剔除密钥键,确认后合并进偏好(同名项被覆盖,之后可逐项恢复默认)。
func (a *App) importSettingsFile(path string) {
	if a.prefs == nil {
		return
	}
	a.bg("read-settings", func() func() {
		defer diag.SpanSlow("settings", "读取设置备份", 300*time.Millisecond)()
		raw, err := readSettingsBackup(path)
		return func() {
			if err != nil {
				a.toast = "无法导入设置:" + err.Error()
				setLog.Warn("设置备份无效", "path", path, "err", err)
				return
			}
			clean, dropped := stripSecretPrefs(raw)
			if len(clean) == 0 {
				a.toast = "这个备份里没有可导入的偏好"
				return
			}
			body := "将合并 " + itoa(len(clean)) + " 项偏好(来自 " + filepath.Base(path) + "),同名的会被覆盖;不含密钥。"
			if dropped > 0 {
				body += "另有 " + itoa(dropped) + " 项疑似密钥的键已忽略。"
			}
			a.confirmDlg("导入设置备份?", body, "导入", false, func() { a.applySettingsBackup(path, clean) })
		}
	})
}

func readSettingsBackup(path string) (map[string]json.RawMessage, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if st.Size() > maxSettingsFile {
		return nil, bigErr{}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, errNotSettings
	}
	return raw, nil
}

type notSettingsErr struct{}

func (notSettingsErr) Error() string { return "不是有效的设置备份(需要 JSON 对象)" }

var errNotSettings = notSettingsErr{}

func (a *App) applySettingsBackup(path string, clean map[string]json.RawMessage) {
	nb, err := json.Marshal(clean)
	if err == nil {
		err = a.prefs.ImportPatch(nb)
	}
	if err != nil {
		a.toast = "合并失败:" + err.Error()
		setLog.Warn("合并设置失败", "path", path, "err", err)
		return
	}
	a.MigrateSettingsPrefs()
	a.applyPrefs()
	a.toast = "已导入 " + itoa(len(clean)) + " 项设置"
	setLog.Info("导入设置", "path", path, "keys", len(clean))
}
