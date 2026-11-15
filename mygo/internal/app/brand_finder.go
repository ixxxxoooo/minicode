package app

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

// appBundlePath 当前进程所在的 .app 包路径;不是从 .app 里启动(go run / 测试)返回空。
func appBundlePath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	p := filepath.Join(filepath.Dir(exe), "..", "..")
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	if strings.HasSuffix(p, ".app") {
		return p
	}
	return ""
}

// syncFinderIcon「同时更换访达里的图标」(appearance.fileIcon):开启时把当前样式写成 .app 包在访达里的自定义图标,
// 关闭时清除。只在「开关 + 样式 + 外观」变化时才动手(签名缓存),后台执行,失败只写日志并 toast 一次。
func (a *App) syncFinderIcon(dark bool) {
	if a.prefs == nil || a.secondary {
		return
	}
	on := a.prefs.Bool(pFileIcon, false)
	suffix, _ := a.dockIconSuffix(dark)
	sig := "off"
	if on {
		sig = "on|" + itoa(a.prefs.Int(pAppStyle, 0)) + suffix
	}
	if sig == a.brand.finderSig || (sig == "off" && a.brand.finderSig == "") {
		a.brand.finderSig = sig
		return
	}
	bundle := appBundlePath()
	if bundle == "" {
		a.brand.finderSig = sig
		return
	}
	a.brand.finderSig = sig
	var png []byte
	if on {
		png = a.dockIconData(dark)
	}
	a.bg("finder-icon", func() func() {
		defer diag.SpanSlow("brand", "更换访达图标", 2*time.Second)()
		ok := setFinderIcon(bundle, png)
		return func() {
			if !ok {
				brandLog.Warn("更换访达图标失败(包目录不可写或无权限)", "bundle", bundle, "on", on)
				a.toast = "访达图标没有更换成功:应用包所在位置不可写"
				return
			}
			brandLog.Info("访达图标已更新", "bundle", bundle, "on", on)
		}
	})
}
