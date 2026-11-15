package app

import (
	"os"
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
)

// 本地图片的解码缓存:界面线程只查表,读盘与解码放后台,避免在构建界面时阻塞 IO。
var imgCache = struct {
	sync.Mutex
	src     map[string]ui.ImageSource
	loading map[string]bool
	failed  map[string]bool
}{src: map[string]ui.ImageSource{}, loading: map[string]bool{}, failed: map[string]bool{}}

// localImage 返回路径对应的图片源;还没解码完(或失败)返回 nil,调用方显示占位。
// 第一次请求会在后台读盘解码,完成后经 a.bg 回到主线程触发重绘。
func (a *App) localImage(path string) ui.ImageSource {
	imgCache.Lock()
	if s, ok := imgCache.src[path]; ok {
		imgCache.Unlock()
		return s
	}
	if imgCache.failed[path] || imgCache.loading[path] {
		imgCache.Unlock()
		return nil
	}
	imgCache.loading[path] = true
	imgCache.Unlock() // 必须先解锁:a.win == nil 时 a.bg 会同步执行下面的闭包

	a.bg("img-load", func() func() {
		data, err := os.ReadFile(path)
		var src ui.ImageSource
		if err == nil {
			src, err = ui.DecodeImage(data)
		}
		imgCache.Lock()
		delete(imgCache.loading, path)
		if err != nil || src == nil {
			imgCache.failed[path] = true
			imgCache.Unlock()
			diag.L("app").Warn("图片解码失败", "path", path, "err", err)
			return nil
		}
		imgCache.src[path] = src
		imgCache.Unlock()
		return func() {}
	})

	imgCache.Lock()
	defer imgCache.Unlock()
	return imgCache.src[path] // 同步执行时这里已经有结果,否则为 nil
}
