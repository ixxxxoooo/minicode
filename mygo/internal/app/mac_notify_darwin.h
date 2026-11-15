#ifndef MAC_NOTIFY_DARWIN_H
#define MAC_NOTIFY_DARWIN_H

void mc_install_notify_delegate(int *inAppBundle);
void mc_post_notify(const char *title, const char *body, const char *sess);

#endif
