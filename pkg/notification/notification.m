#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>

#import "notification.h"

@interface NotificationDelegate : NSObject <NSUserNotificationCenterDelegate>
@end

@implementation NSBundle (FakeBundleIdentifier)
- (NSString *)__bundleIdentifier {
  if (self == [NSBundle mainBundle]) {
    return @"com.github.mtougeron.oncall-status";
  } else {
    return [self __bundleIdentifier];
  }
}
@end

static BOOL installBundleIdentifierHook() {
  Class class = objc_getClass("NSBundle");
  if (class) {
    method_exchangeImplementations(class_getInstanceMethod(class, @selector(bundleIdentifier)), class_getInstanceMethod(class, @selector(__bundleIdentifier)));
    return YES;
  }
  return NO;
}

void showNotification(const char *jsonString) {
	installBundleIdentifierHook();

	NSDictionary *jsonDict = [NSJSONSerialization
	                          JSONObjectWithData:[[NSString stringWithUTF8String:jsonString]
	                                              dataUsingEncoding:NSUTF8StringEncoding]
	                          options:0
	                          error:nil];
	NSUserNotification *notification = [NSUserNotification new];
	BOOL showsButtons = NO;
	notification.title = jsonDict[@"Title"];
	notification.subtitle = jsonDict[@"Subtitle"];
	notification.informativeText = jsonDict[@"Message"];
	NSString *identifier = jsonDict[@"Identifier"];
	if (identifier.length > 0) {
		notification.identifier = identifier;
	}

	NSString *url = jsonDict[@"URL"];
	if (url.length > 0) {
		NSMutableDictionary *options = [NSMutableDictionary dictionary];
		options[@"URL"] = url;
		notification.userInfo = options;
	}

	NSString *closeButton = jsonDict[@"CloseButton"];
	if (closeButton.length > 0) {
		showsButtons = true;
		notification.otherButtonTitle = closeButton;
	}
	NSString *actionButton = jsonDict[@"ActionButton"];
	if (actionButton.length > 0) {
		showsButtons = true;
		notification.actionButtonTitle = actionButton;
	}
	NSString *responsePlaceholder = jsonDict[@"ResponsePlaceholder"];
	if (responsePlaceholder.length > 0) {
		notification.hasReplyButton = YES;
		notification.responsePlaceholder = responsePlaceholder;
	}
	if (showsButtons) {
		// Override banner setting, could check plist to see if we're already set to alerts
		[notification setValue:@YES forKey:@"_showsButtons"];
	}
	BOOL removeFromNotificationCenter = [jsonDict[@"RemoveFromNotificationCenter"] boolValue];
	dispatch_async(dispatch_get_main_queue(), ^{
		NSUserNotificationCenter *center =
			[NSUserNotificationCenter defaultUserNotificationCenter];

		NotificationDelegate *delegate = [[NotificationDelegate alloc] init];
		center.delegate = delegate;
		
		[center deliverNotification:notification];
		if (removeFromNotificationCenter) {
		        [center removeDeliveredNotification:notification];
		}
	});
}

@implementation NotificationDelegate
- (BOOL)userNotificationCenter:(NSUserNotificationCenter *)center shouldPresentNotification:(NSUserNotification *)userNotification {
  return YES;
}

- (void)userNotificationCenter:(NSUserNotificationCenter *)center didActivateNotification:(NSUserNotification *)userNotification {
  // There is no easy way to determine if close button was clicked
  // https://stackoverflow.com/questions/21110714/mac-os-x-nsusernotificationcenter-notification-get-dismiss-event-callback
  switch (userNotification.activationType) {
    case NSUserNotificationActivationTypeAdditionalActionClicked:
	case NSUserNotificationActivationTypeContentsClicked:
    case NSUserNotificationActivationTypeActionButtonClicked: {
		NSString *url     = userNotification.userInfo[@"URL"];
		if (url) [[NSWorkspace sharedWorkspace] openURL:[NSURL URLWithString:url]];
		break;
	}
    case NSUserNotificationActivationTypeReplied:
      break;
    case NSUserNotificationActivationTypeNone:
	  break;
  }

  [[NSUserNotificationCenter defaultUserNotificationCenter] removeDeliveredNotification:userNotification];
}

@end
