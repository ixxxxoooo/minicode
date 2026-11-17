package kit

import "time"

// 动效时钟:所有「按时间推进」的动效(换内容淡入、列表逐行落下、面板收起保留、形态切换进度……)
// 都读这里的时间,而不是直接 time.Now()。
// 真实窗口里恒等于系统时钟;无窗口测试 / cmd/shot 逐帧追踪时注入 Tester 的时钟,
// 这样「推进 16 ms」就真的让这些动效前进 16 ms,逐帧曲线可复现(不再依赖墙钟与机器快慢)。
var clock func() time.Time

// SetClock 注入动效时钟(仅测试 / 截图用);传 nil 恢复系统时钟。
func SetClock(f func() time.Time) { clock = f }

// Now 是动效时钟的当前时间。
func Now() time.Time {
	if clock != nil {
		return clock()
	}
	return time.Now()
}

// Since 是动效时钟下距 t 过了多久。
func Since(t time.Time) time.Duration { return Now().Sub(t) }
