package app

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/egoist/mygo/ui"
)

// 已安装字体的文件名清单(扫用户 / 全局字体目录)与系统字体族列表(后台 ui.FontFamilies)。
var fontScan struct {
	once  sync.Once
	done  atomic.Bool
	files []string // 规范化后的文件名(小写、去空格 / 连字符 / 下划线)
	raw   []string // 小写原文件名
}

var systemFontScan struct {
	once    sync.Once
	done    atomic.Bool
	fams    []ui.FontFamily
	byLower map[string]ui.FontFamily
}

// normFont 把字体名 / 文件名规范化,方便「族名 ⊂ 文件名」比较。
func normFont(s string) string {
	s = strings.ToLower(s)
	return strings.NewReplacer(" ", "", "-", "", "_", "").Replace(s)
}

// startFontScan 第一次调用时在后台扫字体目录,扫完让终端字体重新套用一次。
func (a *App) startFontScan() {
	fontScan.once.Do(func() {
		a.bg("扫描已安装字体", func() func() {
			home, _ := os.UserHomeDir()
			dirs := []string{"/Library/Fonts", "/System/Library/Fonts"}
			if home != "" {
				dirs = append(dirs, filepath.Join(home, "Library", "Fonts"))
			}
			var files, raw []string
			for _, d := range dirs {
				ents, err := os.ReadDir(d)
				if err != nil {
					continue
				}
				for _, e := range ents {
					n := e.Name()
					raw = append(raw, strings.ToLower(n))
					files = append(files, normFont(n))
				}
			}
			fontScan.files, fontScan.raw = files, raw
			fontScan.done.Store(true)
			log.Info("已扫描字体目录", "files", len(files), "nerd", nerdFontInstalled())
			return func() { a.applyPrefs() }
		})
	})
	a.startSystemFontScan()
}

// startSystemFontScan 后台枚举系统字体族(MyGo CoreText / 平台字体列表),供设置选择器使用。
func (a *App) startSystemFontScan() {
	if runtime.GOOS != "darwin" {
		return
	}
	systemFontScan.once.Do(func() {
		a.bg("枚举系统字体", func() func() {
			fams := ui.FontFamilies()
			by := make(map[string]ui.FontFamily, len(fams))
			for _, f := range fams {
				by[strings.ToLower(f.Name)] = f
			}
			systemFontScan.fams, systemFontScan.byLower = fams, by
			systemFontScan.done.Store(true)
			log.Info("系统字体枚举完成", "families", len(fams))
			return func() { a.redraw() }
		})
	})
}

// fontInstalled 判断字体族是否装了:installed 为结果,known 表示扫描已完成(未完成时 installed 无意义)。
func fontInstalled(family string) (installed, known bool) {
	if family == "" || strings.EqualFold(family, "monospace") {
		return true, true
	}
	// macOS:直接问 CoreText,即时且准确(装完字体立刻能认出来,不用等目录重新扫描)
	if _, inst, known := fontHas(family, glyphCats[0]); known {
		return inst, true
	}
	sysDone := systemFontScan.done.Load()
	fileDone := fontScan.done.Load()
	if !sysDone && !fileDone {
		return false, false
	}
	if sysDone {
		if _, ok := systemFontScan.byLower[strings.ToLower(family)]; ok {
			return true, true
		}
	}
	if fileDone {
		want := normFont(family)
		for _, f := range fontScan.files {
			if strings.Contains(f, want) {
				return true, true
			}
		}
	}
	if sysDone && fileDone {
		return false, true
	}
	return false, false
}

// fontInstalledLabel 返回「已安装 / 未安装 / 检测中」。
func fontInstalledLabel(family string) (label string, known bool, ok bool) {
	if family == "" {
		return "系统等宽", true, true
	}
	ok, known = fontInstalled(family)
	switch {
	case !known:
		return "检测中…", false, false
	case ok:
		return "已安装", true, true
	default:
		return "未安装", true, false
	}
}

// nerdFontInstalled 是否装了任何 Nerd Font(按文件名认:含 nerd / meslolgs / 「NF」后缀)。
func nerdFontInstalled() bool {
	if !fontScan.done.Load() {
		return false
	}
	for _, f := range fontScan.raw {
		if strings.Contains(f, "nerd") || strings.Contains(f, "meslolgs") ||
			strings.Contains(f, " nf") || strings.Contains(f, "-nf") || strings.Contains(f, "nfm") {
			return true
		}
	}
	return false
}

// systemMonoFamilies 返回系统里标记为等宽的字体族(未枚举完时返回 nil)。
func systemMonoFamilies() []ui.FontFamily {
	if !systemFontScan.done.Load() {
		return nil
	}
	var out []ui.FontFamily
	for _, f := range systemFontScan.fams {
		if f.Monospaced {
			out = append(out, f)
		}
	}
	return out
}

// resetFontScans 让字体目录与系统字体枚举在下次使用时重新来一遍(装了新字体之后)。
func resetFontScans() {
	fontScan.once, systemFontScan.once = sync.Once{}, sync.Once{}
	fontScan.done.Store(false)
	systemFontScan.done.Store(false)
	resetFontCoverage()
}
