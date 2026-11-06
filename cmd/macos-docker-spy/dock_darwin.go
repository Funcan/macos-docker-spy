package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

static void hideFromDock(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		[NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
	});
}
*/
import "C"

// hideFromDock makes this a menu bar only app, with no Dock icon or app menu,
// while still allowing windows. Fyne's windowing library makes it a regular
// app as it starts up, so call this once the app has started.
func hideFromDock() {
	C.hideFromDock()
}
