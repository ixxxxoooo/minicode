// Package backend 定义界面与「Agent 后端」之间的唯一接口。
// 演示后端(internal/demo)与真实 claude 后端(internal/claude)实现同一个接口,所以界面一份代码两种用法。
package backend

import "github.com/ixxxxoooo/minicode/internal/core"

// Sink 接收后端产出的事件。后端可在任意 goroutine 调用它;界面侧的实现负责切回主线程再归约。
type Sink func(core.Event)

// Backend 是一个会话的后端。
type Backend interface {
	// Name 用于日志与「运行模式」徽章:demo / claude。
	Name() string
	// Start 开始产出事件(可能是惰性的:真实后端到第一条提示才起进程)。只调一次。
	Start(sink Sink)
	// Send 发送界面命令。不阻塞。
	Send(cmd core.Command)
	// Close 释放资源(结束子进程、停掉定时器)。
	Close()
}
