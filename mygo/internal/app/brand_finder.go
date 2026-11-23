package app

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/macos"
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

// 访达图标的几种结果(设置页「当前生效」一行直接读它)。
const (
	finderOff     = "未开启"
	finderNoApp   = "不是从 .app 启动,没有可更换的包"
	finderWorking = "正在更换…"
)

// syncFinderIcon「同时更换访达里的图标」(appearance.fileIcon):开启时把当前样式写成 .app 包在访达里的自定义图标,
// 关闭时清除。只在「开关 + 样式 + 外观」变化时才动手(签名缓存),后台执行;
// 失败时写日志、toast 一次,并在设置页写明原因;换样式 / 再点开关 / 点「重新应用」都会重试。
//
// 实现:NSWorkspace.setIcon:forFile: 在 .app 根目录写一个隐藏的 Icon\r 文件(资源叉里放图标)。
// 它会让 `codesign --verify --strict` 报「resource fork … not allowed」,但不影响启动与更新(更新是整包替换,
// 新包没有这个文件,所以更新后第一次启动会自动重新套用)。关闭开关即删除该文件,签名恢复。
func (a *App) syncFinderIcon(dark bool) {
	if a.prefs == nil || a.secondary {
		return
	}
	on := a.prefs.Bool(pFileIcon, false)
	suffix, _ := a.dockIconSuffix(dark)
	sig := "off"
	if on {
		sig = "on|" + itoa(a.iconStyleIndex()) + "|" + a.appIconPal() + suffix
	}
	bundle := appBundlePath()
	if sig == a.brand.finderSig || (sig == "off" && a.brand.finderSig == "") {
		a.brand.finderSig = sig
		if !on {
			a.brand.finderState = finderOff
		}
		return
	}
	if bundle == "" {
		a.brand.finderSig = sig
		a.brand.finderState = finderNoApp
		if on {
			brandLog.Warn("访达图标没有更换:当前进程不是从 .app 包启动的", "exe", os.Args[0])
		}
		return
	}
	a.brand.finderSig = sig
	var png []byte
	if on {
		png = a.dockIconData(dark)
		a.brand.finderState = finderWorking
	} else {
		a.brand.finderState = finderOff
	}
	a.bg("finder-icon", func() func() {
		defer diag.SpanSlow("brand", "更换访达图标", 2*time.Second)()
		ok := macos.SetFinderIcon(bundle, png)
		if ok {
			// 碰一下包目录的修改时间,提醒访达与启动台重新读取图标(系统的图标缓存有时不会立刻刷新)
			now := time.Now()
			_ = os.Chtimes(bundle, now, now)
		}
		_, statErr := os.Stat(filepath.Join(bundle, "Icon\r"))
		written := statErr == nil
		return func() {
			switch {
			case !ok:
				// 失败后保留签名,避免每次事件同步都重试刷屏;换样式 / 再点开关 / 点「重新应用」会重新尝试
				if on {
					a.brand.finderState = "失败:应用包所在位置不可写或没有权限"
					a.toast = "访达图标没有更换成功:应用包所在位置不可写(可把应用放到「应用程序」后重试)"
				}
				brandLog.Warn("更换访达图标失败(包目录不可写或无权限)", "bundle", bundle, "on", on)
			case on && !written:
				a.brand.finderState = "失败:系统没有写入图标文件"
				brandLog.Warn("访达图标调用成功但没有生成 Icon 文件", "bundle", bundle)
			case on:
				a.brand.finderState = "已更换"
				brandLog.Info("访达图标已更新", "bundle", bundle, "style", appStyles[a.iconStyleIndex()])
			default:
				a.brand.finderState = finderOff
				brandLog.Info("访达图标已还原", "bundle", bundle)
			}
		}
	})
}

// reapplyBrand 强制把 Dock 图标、访达图标与菜单栏图标全部重新套用一遍(设置里的「重新应用」按钮),
// 用于「改了设置却好像没生效」时一键兜底;各步结果写进日志。
func (a *App) reapplyBrand() {
	brandLog.Info("手动重新应用图标", "style", appStyles[a.iconStyleIndex()])
	a.brand.dockSig = ""
	a.brand.finderSig = ""
	a.brand.trayMode = ""
	a.brand.lastBadge = ""
	a.syncBrand()
	a.flash("已重新应用 Dock、访达与菜单栏图标")
}
