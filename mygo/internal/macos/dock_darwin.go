//go:build darwin && cgo

package macos

/*
#cgo darwin CFLAGS: -x objective-c -fobjc-arc
#cgo darwin LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

static int dockAttention(int critical) {
	return (int)[NSApp requestUserAttention:(critical ? NSCriticalRequest : NSInformationalRequest)];
}

static void dockCancelAttention(int id) {
	if (id > 0) [NSApp cancelUserAttentionRequest:id];
}

static void dockSetProgress(double frac) {
	NSDockTile *tile = [NSApp dockTile];
	if (frac <= 0 || frac > 1) {
		[tile setContentView:nil];
		[tile display];
		return;
	}
	NSView *view = tile.contentView;
	if (!view) {
		view = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, 128, 128)];
		[tile setContentView:view];
	}
	NSRect b = view.bounds;
	NSRect bar = NSMakeRect(8, 10, b.size.width - 16, 6);
	for (NSView *v in [view.subviews copy]) [v removeFromSuperview];
	NSView *track = [[NSView alloc] initWithFrame:bar];
	track.wantsLayer = YES;
	track.layer.backgroundColor = [[NSColor colorWithWhite:1 alpha:.25] CGColor];
	track.layer.cornerRadius = 3;
	[view addSubview:track];
	NSRect fill = bar;
	fill.size.width *= frac;
	NSView *fillV = [[NSView alloc] initWithFrame:fill];
	fillV.wantsLayer = YES;
	fillV.layer.backgroundColor = [[NSColor systemBlueColor] CGColor];
	fillV.layer.cornerRadius = 3;
	[view addSubview:fillV];
	[tile display];
}
*/
import "C"

// DockRequestAttention 让 Dock 图标弹跳(critical 为 true 则持续弹跳到被激活),返回请求 id,0 表示没发出。
func DockRequestAttention(critical bool) int {
	crit := 0
	if critical {
		crit = 1
	}
	return int(C.dockAttention(C.int(crit)))
}

// DockCancelAttention 取消 DockRequestAttention 发出的弹跳。
func DockCancelAttention(id int) {
	if id > 0 {
		C.dockCancelAttention(C.int(id))
	}
}

// DockSetProgress 在 Dock 图标下方画进度条(frac 取 0~1 之外则清除)。
func DockSetProgress(frac float64) {
	C.dockSetProgress(C.double(frac))
}
