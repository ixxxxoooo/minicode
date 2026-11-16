//go:build darwin && cgo

package kit

/*
#cgo darwin CFLAGS: -x objective-c -fobjc-arc
#cgo darwin LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

static int kitReducedMotion(void) {
	return [NSWorkspace.sharedWorkspace accessibilityDisplayShouldReduceMotion] ? 1 : 0;
}
*/
import "C"

// SystemReduceMotion 读 macOS「减弱动态效果」。
//
// 直接读 NSWorkspace 的无障碍属性(微秒级,可在任意线程调用)。
// 原先用 exec 起 `defaults read com.apple.universalaccess reduceMotion`,每次约 5~8 ms 的 fork/exec,
// 而它被托盘 / Dock 同步按事件频繁调用,打开长会话时在界面线程上累计出数秒停滞。
func SystemReduceMotion() bool { return C.kitReducedMotion() != 0 }
