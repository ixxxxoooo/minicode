#import <Foundation/Foundation.h>
#import <UserNotifications/UserNotifications.h>
#import <AppKit/AppKit.h>

extern void goMCNotifySession(char *sess);
static void mc_go_notify_open(const char *sess);

@interface MCNotifyDelegate : NSObject <UNUserNotificationCenterDelegate>
@end

@implementation MCNotifyDelegate
- (void)userNotificationCenter:(UNUserNotificationCenter *)center
    didReceiveNotificationResponse:(UNNotificationResponse *)response
             withCompletionHandler:(void (^)(void))completionHandler {
  NSDictionary *info = response.notification.request.content.userInfo;
  NSString *sess = info[@"sess"];
  if (sess.length > 0) {
    mc_go_notify_open([sess UTF8String]);
  }
  if (completionHandler) {
    completionHandler();
  }
}
@end

static MCNotifyDelegate *mc_delegate;
static BOOL mc_bundle_app;

static BOOL mc_running_in_app_bundle(void) {
  NSBundle *b = [NSBundle mainBundle];
  if (!b || !b.bundlePath) {
    return NO;
  }
  return [b.bundlePath hasSuffix:@".app"];
}

void mc_install_notify_delegate(int *inAppBundle) {
  mc_bundle_app = mc_running_in_app_bundle();
  if (inAppBundle) {
    *inAppBundle = mc_bundle_app ? 1 : 0;
  }
  if (!mc_bundle_app) {
    return;
  }
  if (@available(macOS 10.14, *)) {
    if (!mc_delegate) {
      mc_delegate = [MCNotifyDelegate new];
    }
    UNUserNotificationCenter *c = [UNUserNotificationCenter currentNotificationCenter];
    c.delegate = mc_delegate;
    [c requestAuthorizationWithOptions:(UNAuthorizationOptionAlert | UNAuthorizationOptionSound)
                     completionHandler:^(BOOL granted, NSError * _Nullable error) {
      if (error) {
        NSLog(@"MiniCode notify auth: %@", error);
      } else if (!granted) {
        NSLog(@"MiniCode notify auth denied");
      }
    }];
  }
}

static void mc_go_notify_open(const char *sess) {
  goMCNotifySession((char *)sess);
}

void mc_post_notify(const char *title, const char *body, const char *sess, int playSound) {
  if (@available(macOS 10.14, *)) {
    if (!mc_bundle_app) {
      return;
    }
    UNMutableNotificationContent *c = [UNMutableNotificationContent new];
    c.title = [NSString stringWithUTF8String:title];
    c.body = [NSString stringWithUTF8String:body];
    if (playSound) {
      c.sound = [UNNotificationSound defaultSound];
    }
    if (sess && sess[0]) {
      c.userInfo = @{@"sess": [NSString stringWithUTF8String:sess]};
      c.categoryIdentifier = @"minicode.session";
    }
    NSString *nid = [NSString stringWithFormat:@"minicode-%f", [NSDate timeIntervalSinceReferenceDate]];
    UNNotificationRequest *req = [UNNotificationRequest requestWithIdentifier:nid content:c trigger:nil];
    [[UNUserNotificationCenter currentNotificationCenter] addNotificationRequest:req
        withCompletionHandler:^(NSError * _Nullable error) {
      if (error) {
        NSLog(@"MiniCode notify: %@", error);
      }
    }];
  }
}
