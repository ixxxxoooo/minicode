//go:build darwin

package app

/*
#cgo darwin LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
static int isDark(void) {
	NSString *name = [[NSApp effectiveAppearance] name];
	return [name containsString:@"Dark"] ? 1 : 0;
}
*/
import "C"

func systemIsDark() bool { return C.isDark() != 0 }
