//go:build darwin && cgo

package app

/*
#cgo darwin CFLAGS: -x objective-c -fobjc-arc
#cgo darwin LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

static int a11yReducedTransparency(void) {
	return [NSWorkspace.sharedWorkspace accessibilityDisplayShouldReduceTransparency] ? 1 : 0;
}
static int a11yIncreasedContrast(void) {
	return [NSWorkspace.sharedWorkspace accessibilityDisplayShouldIncreaseContrast] ? 1 : 0;
}
static int a11yReducedMotion(void) {
	return [NSWorkspace.sharedWorkspace accessibilityDisplayShouldReduceMotion] ? 1 : 0;
}

static void postA11yAnnouncement(const char *utf8) {
	if (!utf8 || !*utf8) return;
	NSString *text = [NSString stringWithUTF8String:utf8];
	if (text.length == 0) return;
	NSDictionary *info = @{
		NSAccessibilityAnnouncementKey: text,
		NSAccessibilityPriorityKey: @(NSAccessibilityPriorityHigh),
	};
	NSAccessibilityPostNotificationWithUserInfo(
		NSApp, NSAccessibilityAnnouncementRequestedNotification, info);
}
*/
import "C"
import "unsafe"

func a11yReduceTransparency() bool { return C.a11yReducedTransparency() != 0 }
func a11yIncreaseContrast() bool   { return C.a11yIncreasedContrast() != 0 }
func a11yReduceMotion() bool       { return C.a11yReducedMotion() != 0 }

func postA11yAnnouncement(text string) {
	if text == "" {
		return
	}
	cs := C.CString(text)
	defer C.free(unsafe.Pointer(cs))
	C.postA11yAnnouncement(cs)
}
