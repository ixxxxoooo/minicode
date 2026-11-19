package app

import (
	"reflect"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// listDirAsync 在后台读一个目录,读完回主线程写进目录缓存(ide.listing 的异步一半)。
//
// 为什么:此前缓存失效(刷新 / 文件变化 / 换项目)后,界面线程在下一帧同步 os.ReadDir,磁盘慢或系统弹权限提示时
// 整个界面冻住,正在播放的 Agent ⇄ IDE 切换动画停在半路的某一帧(主线程停滞实测 0.4 ~ 9 秒,栈在 os.ReadDir)。
// 项目约定「界面线程不做阻塞 IO」,这里改为后台读;读的期间 listing 返回上一次读到的列表,树不闪空。
//
// 参数 dir:要读的目录。无返回值。读取结果若发现缓存在途中被清空(缓存换了一代),就丢弃,下一帧会重新请求。
func (a *App) listDirAsync(dir string) {
	d := a.ide
	gen := d.listCache // 请求时的缓存代:映射的指针
	fs := d.fs
	a.bg("列目录", func() func() {
		t0 := time.Now()
		es, err := fs.List(dir)
		if took := time.Since(t0); took > 300*time.Millisecond {
			diag.L("ide").Warn("列目录很慢", "dir", dir, "耗时", took) // 磁盘慢 / 权限提示:此前这里会冻住界面线程
		}
		return func() {
			delete(d.listPend, dir)
			if err != nil {
				diag.L("ide").Warn("列目录失败", "dir", dir, "err", err)
			}
			if d.listCache == nil || gen == nil || reflect.ValueOf(d.listCache).Pointer() != reflect.ValueOf(gen).Pointer() {
				return // 读的途中缓存被清空了:结果可能已过期,丢弃,下一帧重新请求
			}
			es = d.visible(es) // 按「显示隐藏文件」过滤后再进缓存
			d.listCache[dir] = es
			d.treeLineBuf = nil // 新列表到了:拍平的树行要重算,否则缓存了读之前的占位内容
			if d.listLast == nil {
				d.listLast = map[string][]workspace.Entry{}
			}
			d.listLast[dir] = es
		}
	})
}
