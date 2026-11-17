#ifndef MAC_NOTIFY_DARWIN_H
#define MAC_NOTIFY_DARWIN_H

void mc_install_notify_delegate(int *inAppBundle);
// kind:0 普通 / 1 需要确认(带「允许一次 / 拒绝」动作,时效性)/ 2 出错(时效性)
void mc_post_notify(const char *title, const char *body, const char *sess, const char *permID, int kind, int playSound, const char *iconPath);
// 撤掉某个确认请求的横幅(在应用里处理完了,横幅上的按钮不该再能点)
void mc_clear_notify(const char *permID);

#endif
