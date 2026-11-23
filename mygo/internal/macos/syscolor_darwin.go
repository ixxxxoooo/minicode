//go:build darwin && cgo

package macos

/*
#cgo darwin LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
static int isDark(void) {
	NSString *name = [[NSApp effectiveAppearance] name];
	return [name containsString:@"Dark"] ? 1 : 0;
}
*/
import "C"

// SystemIsDark 系统当前是否是深色外观。
func SystemIsDark() bool { return C.isDark() != 0 }
