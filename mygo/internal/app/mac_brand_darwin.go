//go:build darwin && cgo

package app

/*
#cgo darwin CFLAGS: -x objective-c -fobjc-arc
#cgo darwin LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#include <stdlib.h>
#include <string.h>

static int exportPNGFromIcns(const char *path, unsigned char **out, size_t *outLen) {
	@autoreleasepool {
		NSString *p = [NSString stringWithUTF8String:path];
		NSImage *img = [[NSImage alloc] initWithContentsOfFile:p];
		if (!img) return -1;
		[img setSize:NSMakeSize(512, 512)];
		NSRect rect = NSMakeRect(0, 0, 512, 512);
		NSImageRep *imgRep = [img bestRepresentationForRect:rect context:nil hints:nil];
		if (!imgRep || ![imgRep isKindOfClass:[NSBitmapImageRep class]]) return -2;
		NSBitmapImageRep *rep = (NSBitmapImageRep *)imgRep;
		NSData *data = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
		if (!data || data.length == 0) return -3;
		*outLen = (size_t)data.length;
		*out = (unsigned char *)malloc(*outLen);
		if (!*out) return -4;
		memcpy(*out, data.bytes, *outLen);
		return 0;
	}
}

static void freeBuf(unsigned char *p) { if (p) free(p); }
*/
import "C"
import (
	"fmt"
	"unsafe"
)

func loadIcnsAsPNG(path string) ([]byte, bool) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	var out *C.uchar
	var n C.size_t
	if C.exportPNGFromIcns(cpath, &out, &n) != 0 {
		return nil, false
	}
	defer C.freeBuf(out)
	if out == nil || n == 0 {
		return nil, false
	}
	b := C.GoBytes(unsafe.Pointer(out), C.int(n))
	if len(b) == 0 {
		return nil, false
	}
	_ = fmt.Sprintf // 保留以便日后写日志
	return b, true
}
