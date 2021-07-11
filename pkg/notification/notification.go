package notification

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>

#ifndef __NOTIFICATION_H_H__
#import "notification.h"
#endif

void showNotification(const char *jsonString);

*/
import "C"
import (
	"encoding/json"
	"log"
	"unsafe"
)

// Notification represents an NSUserNotification
type Notification struct {
	Title    string
	Subtitle string
	Message  string

	// These add an optional action button, change what the close button says, and adds an in-line reply
	ActionButton        string
	CloseButton         string
	ResponsePlaceholder string

	URL string

	// Duplicate identifiers do not re-display, but instead update the notification center
	Identifier string

	// If true, the notification is shown, but then deleted from the notification center
	RemoveFromNotificationCenter bool
}

// ShowNotification shows a notification to the user.
func ShowNotification(notification Notification) {
	b, err := json.Marshal(notification)
	if err != nil {
		log.Printf("Marshal: %v", err)
		return
	}
	cstr := C.CString(string(b))
	C.showNotification(cstr)
	C.free(unsafe.Pointer(cstr)) // #nosec G103
}
