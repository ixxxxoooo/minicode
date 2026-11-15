package app

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

// 已安装字体的文件名清单(只扫用户 / 全局字体目录;Nerd Font 都装在这里,系统自带字体不会是 Nerd Font)。
// 扫描只做一次,放后台;没扫完时按「未知」处理,不替用户换字体。
var fontScan struct {
	once  sync.Once
	done  atomic.Bool
	files []string // 规范化后的文件名(小写、去空格 / 连字符 / 下划线)
	raw   []string // 小写原文件名
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
			dirs := []string{"/Library/Fonts"}
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
}

// fontInstalled 判断字体族是否装了:installed 为结果,known 表示扫描已完成(未完成时 installed 无意义)。
func fontInstalled(family string) (installed, known bool) {
	if !fontScan.done.Load() {
		return false, false
	}
	want := normFont(family)
	if want == "" {
		return false, true
	}
	for _, f := range fontScan.files {
		if strings.Contains(f, want) {
			return true, true
		}
	}
	return false, true
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
