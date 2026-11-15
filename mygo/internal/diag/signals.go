package diag

import (
	"os"
	"os/signal"
	"syscall"
)

// WatchSignals 把 SIGTERM / SIGINT / SIGHUP 当成正常退出:系统关机 / 登出、脚本 `kill`、
// 开发版重启都会发这些信号,不处理的话下次启动会把它们统统记成「未正常退出」,崩溃列表被误报淹没。
// 收到后先写「正常退出」并删运行标记,再以 128+信号号 退出。SIGKILL / 真崩溃仍留标记,照常被识别。
func WatchSignals() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	go func() {
		defer Recover("信号监听")
		sig := <-ch
		CleanExit("收到信号 " + sig.String())
		code := 1
		if s, ok := sig.(syscall.Signal); ok {
			code = 128 + int(s)
		}
		os.Exit(code)
	}()
}
