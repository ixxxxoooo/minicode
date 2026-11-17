#import <Foundation/Foundation.h>
#import <UserNotifications/UserNotifications.h>
#import <AppKit/AppKit.h>

extern void goMCNotifySession(char *sess);
extern void goMCNotifyAction(char *sess, char *permID, char *action);
extern void goMCNotifyDenied(void);
static void mc_go_notify_open(const char *sess);

@interface MCNotifyDelegate : NSObject <UNUserNotificationCenterDelegate>
@end

@implementation MCNotifyDelegate
// 应用在前台时系统默认不展示通知;这里让它照常弹横幅并响提示音。
- (void)userNotificationCenter:(UNUserNotificationCenter *)center
       willPresentNotification:(UNNotification *)notification
         withCompletionHandler:(void (^)(UNNotificationPresentationOptions))completionHandler {
  completionHandler(UNNotificationPresentationOptionBanner | UNNotificationPresentationOptionList |
                    UNNotificationPresentationOptionSound);
}

- (void)userNotificationCenter:(UNUserNotificationCenter *)center
    didReceiveNotificationResponse:(UNNotificationResponse *)response
             withCompletionHandler:(void (^)(void))completionHandler {
  NSDictionary *info = response.notification.request.content.userInfo;
  NSString *sess = info[@"sess"];
  NSString *perm = info[@"perm"];
  NSString *act = response.actionIdentifier;
  if (([act isEqualToString:@"allow"] || [act isEqualToString:@"deny"]) && perm.length > 0) {
    // 通知上的「允许一次 / 拒绝」:不必切回应用就能处理
    goMCNotifyAction((char *)[(sess ?: @"") UTF8String], (char *)[perm UTF8String], (char *)[act UTF8String]);
  } else if (sess.length > 0) {
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
    // 「需要确认」类通知带两个动作;前台才执行的动作不设 Foreground,点了直接在后台处理
    UNNotificationAction *allow = [UNNotificationAction actionWithIdentifier:@"allow" title:@"允许一次" options:UNNotificationActionOptionNone];
    UNNotificationAction *deny = [UNNotificationAction actionWithIdentifier:@"deny" title:@"拒绝" options:UNNotificationActionOptionDestructive];
    UNNotificationCategory *ask = [UNNotificationCategory categoryWithIdentifier:@"minicode.approval" actions:@[ allow, deny ]
                                                              intentIdentifiers:@[] options:UNNotificationCategoryOptionNone];
    UNNotificationCategory *sessCat = [UNNotificationCategory categoryWithIdentifier:@"minicode.session" actions:@[]
                                                                   intentIdentifiers:@[] options:UNNotificationCategoryOptionNone];
    [c setNotificationCategories:[NSSet setWithObjects:ask, sessCat, nil]];
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

void mc_post_notify(const char *title, const char *body, const char *sess, const char *permID, int kind, int playSound, const char *iconPath) {
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
    NSMutableDictionary *info = [NSMutableDictionary dictionary];
    NSString *sessStr = (sess && sess[0]) ? [NSString stringWithUTF8String:sess] : @"";
    NSString *permStr = (permID && permID[0]) ? [NSString stringWithUTF8String:permID] : @"";
    if (sessStr.length > 0) {
      info[@"sess"] = sessStr;
      c.threadIdentifier = sessStr; // 同一会话的通知在通知中心里叠成一组
    }
    if (permStr.length > 0) {
      info[@"perm"] = permStr;
    }
    c.userInfo = info;
    if (iconPath && iconPath[0]) {
      // 附上当前样式的应用图标(通知右侧缩略图);系统会把附件文件搬走,所以先复制一份临时文件
      NSString *src = [NSString stringWithUTF8String:iconPath];
      NSString *tmp = [NSTemporaryDirectory() stringByAppendingPathComponent:
                          [NSString stringWithFormat:@"minicode-notify-%f.png", [NSDate timeIntervalSinceReferenceDate]]];
      NSError *cpErr = nil;
      if ([[NSFileManager defaultManager] copyItemAtPath:src toPath:tmp error:&cpErr]) {
        NSError *atErr = nil;
        UNNotificationAttachment *att = [UNNotificationAttachment attachmentWithIdentifier:@"icon"
                                                                                       URL:[NSURL fileURLWithPath:tmp]
                                                                                   options:nil error:&atErr];
        if (att) {
          c.attachments = @[ att ];
        } else {
          NSLog(@"MiniCode notify icon: %@", atErr);
        }
      }
    }
    c.categoryIdentifier = (kind == 1 && permStr.length > 0) ? @"minicode.approval" : @"minicode.session";
    if (kind >= 1) {
      // 时效性:需要你拍板 / 出错的通知可以穿透专注模式(没有该权限的系统上等同普通通知)
      if (@available(macOS 12.0, *)) {
        c.interruptionLevel = UNNotificationInterruptionLevelTimeSensitive;
      }
    }
    NSString *nid = (kind == 1 && permStr.length > 0) ? [@"minicode-ask-" stringByAppendingString:permStr]
                                                      : [NSString stringWithFormat:@"minicode-%f", [NSDate timeIntervalSinceReferenceDate]];
    UNNotificationRequest *req = [UNNotificationRequest requestWithIdentifier:nid content:c trigger:nil];
    UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
    [center getNotificationSettingsWithCompletionHandler:^(UNNotificationSettings *_Nonnull s) {
      if (s.authorizationStatus == UNAuthorizationStatusDenied) {
        goMCNotifyDenied(); // 用户在系统设置里关了通知:不能静默吞掉,界面要告诉他去哪里打开
        return;
      }
      [center addNotificationRequest:req withCompletionHandler:^(NSError *_Nullable error) {
        if (error) {
          NSLog(@"MiniCode notify: %@", error);
        }
      }];
    }];
  }
}

void mc_clear_notify(const char *permID) {
  if (@available(macOS 10.14, *)) {
    if (!mc_bundle_app || !permID || !permID[0]) {
      return;
    }
    NSString *nid = [@"minicode-ask-" stringByAppendingString:[NSString stringWithUTF8String:permID]];
    UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
    [center removeDeliveredNotificationsWithIdentifiers:@[ nid ]];
    [center removePendingNotificationRequestsWithIdentifiers:@[ nid ]];
  }
}
