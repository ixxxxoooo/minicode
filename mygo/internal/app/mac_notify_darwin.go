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

//export goMCNotifyAction
func goMCNotifyAction(sess, permID, action *C.char) {
	if macNotifyApp == nil {
		return
	}
	s, p, act := C.GoString(sess), C.GoString(permID), C.GoString(action)
	a := macNotifyApp
	if a.win != nil {
		a.win.Update(func() { a.NotifyAction(s, p, act) })
	} else {
		a.NotifyAction(s, p, act)
	}
}

//export goMCNotifyDenied
func goMCNotifyDenied() {
	if macNotifyApp == nil || macNotifyApp.win == nil {
		return
	}
	a := macNotifyApp
	a.win.Update(func() { a.notifyDenied() })
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

func postMacNotification(n sysNote, sound bool) {
	if macNotifyApp == nil {
		return
	}
	if !macNotifyBundle {
		if sound {
			macNotifyApp.notifyOSAScript(n.title, n.body)
		} else {
			macNotifyApp.notifyOSAScriptSilent(n.title, n.body)
		}
		return
	}
	ct, cb, cs, cp := C.CString(n.title), C.CString(n.body), C.CString(n.sess), C.CString(n.permID)
	defer C.free(unsafe.Pointer(ct))
	defer C.free(unsafe.Pointer(cb))
	defer C.free(unsafe.Pointer(cs))
	defer C.free(unsafe.Pointer(cp))
	snd := 0
	if sound {
		snd = 1
	}
	C.mc_post_notify(ct, cb, cs, cp, C.int(n.kind), C.int(snd))
}

// clearMacNotification 撤掉某个确认请求的系统横幅(已在应用里处理)。
func clearMacNotification(permID string) {
	if !macNotifyBundle || permID == "" {
		return
	}
	cp := C.CString(permID)
	defer C.free(unsafe.Pointer(cp))
	C.mc_clear_notify(cp)
}

var macNotifyBundle bool

func initMacNotifyBundle() {
	if !macNotifyBundle {
		macNotifyBundle = installMacNotifyDelegate()
	}
}
