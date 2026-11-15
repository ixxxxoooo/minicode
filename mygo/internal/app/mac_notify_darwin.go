//go:build darwin && cgo

package app

/*
#cgo darwin CFLAGS: -x objective-c -fobjc-arc
#cgo darwin LDFLAGS: -framework Foundation -framework UserNotifications
#import <Foundation/Foundation.h>
#import <UserNotifications/UserNotifications.h>

static void mc_post_notify(const char *title, const char *body, const char *sess) {
  if (@available(macOS 10.14, *)) {
    UNMutableNotificationContent *c = [UNMutableNotificationContent new];
    c.title = [NSString stringWithUTF8String:title];
    c.body = [NSString stringWithUTF8String:body];
    c.sound = [UNNotificationSound defaultSound];
    if (sess && sess[0]) {
      c.userInfo = @{@"sess": [NSString stringWithUTF8String:sess]};
      c.categoryIdentifier = @"minicode.session";
    }
    UNNotificationRequest *req = [UNNotificationRequest requestWithIdentifier:@"minicode"
        content:c trigger:nil];
    [[UNUserNotificationCenter currentNotificationCenter] addNotificationRequest:req
        withCompletionHandler:^(NSError * _Nullable error) {
      if (error) {
        NSLog(@"MiniCode notify: %@", error);
      }
    }];
  }
}
*/
import "C"
import "unsafe"

func postMacNotification(title, body, sessID string) {
	if macNotifyApp == nil {
		return
	}
	ct, cb, cs := C.CString(title), C.CString(body), C.CString(sessID)
	defer C.free(unsafe.Pointer(ct))
	defer C.free(unsafe.Pointer(cb))
	defer C.free(unsafe.Pointer(cs))
	C.mc_post_notify(ct, cb, cs)
}
