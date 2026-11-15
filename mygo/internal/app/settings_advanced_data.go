package app

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (a *App) exportSettingsJSON() {
	if a.prefs == nil {
		return
	}
	path := filepath.Join(os.TempDir(), "minicode-settings-export.json")
	data, err := a.prefs.ExportJSON()
	if err != nil {
		a.toast = "导出失败"
		return
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		a.toast = "写入失败"
		return
	}
	a.toast = "已导出 settings.json(不含密钥)"
	setLog.Info("导出设置", "path", path)
}

func (a *App) askImportSettings() {
	a.confirmDlg("导入设置", "将合并 ~/.minicode/settings-import.json 中的偏好(不含密钥)。", "导入", false, func() { a.doImportSettings() })
}

func (a *App) doImportSettings() {
	a.toast = "导入:请将 JSON 放到 ~/.minicode/settings-import.json 后重试"
	path := filepath.Join(os.Getenv("HOME"), ".minicode", "settings-import.json")
	b, err := os.ReadFile(path)
	if err != nil {
		setLog.Warn("导入设置找不到文件", "path", path, "err", err)
		return
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(b, &raw) != nil {
		a.toast = "JSON 无效"
		return
	}
	for k := range raw {
		if strings.Contains(k, "key") || strings.Contains(k, "secret") {
			delete(raw, k)
		}
	}
	nb, _ := json.Marshal(raw)
	if err := a.prefs.ImportPatch(nb); err != nil {
		a.toast = "合并失败"
		return
	}
	a.MigrateSettingsPrefs()
	a.applyPrefs()
	a.toast = "已导入设置"
	setLog.Info("导入设置", "path", path)
}

func openURL(url string) {
	_ = exec.Command("open", url).Start()
}
