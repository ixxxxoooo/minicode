//go:build darwin && cgo

package app

/*
#cgo darwin CFLAGS: -x objective-c -fobjc-arc
#cgo darwin LDFLAGS: -framework Foundation -framework UserNotifications -framework AppKit
#include <stdlib.h>
#include "mac_notify_darwin.h"
*/
import "C"
import "unsafe"

//export goMCNotifySession
func goMCNotifySession(sess *C.char) {
	if macNotifyApp == nil {
		return
	}
	sid := C.GoString(sess)
	a := macNotifyApp
	if a.win != nil {
		a.win.Update(func() {
			a.OpenSessionFromNotify(sid)
		})
	} else {
		a.OpenSessionFromNotify(sid)
	}
}

func installMacNotifyDelegate() bool {
	var inApp C.int
	C.mc_install_notify_delegate(&inApp)
	ok := inApp != 0
	if !ok {
		log.Warn("系统通知:未在 .app 包内运行,点击回会话不可用(开发 go run 将降级)")
	}
	return ok
}

func postMacNotification(title, body, sessID string) {
	if macNotifyApp == nil {
		return
	}
	if !macNotifyBundle {
		macNotifyApp.notifyOSAScript(title, body)
		return
	}
	ct, cb, cs := C.CString(title), C.CString(body), C.CString(sessID)
	defer C.free(unsafe.Pointer(ct))
	defer C.free(unsafe.Pointer(cb))
	defer C.free(unsafe.Pointer(cs))
	C.mc_post_notify(ct, cb, cs)
}

var macNotifyBundle bool

func initMacNotifyBundle() {
	if !macNotifyBundle {
		macNotifyBundle = installMacNotifyDelegate()
	}
}
