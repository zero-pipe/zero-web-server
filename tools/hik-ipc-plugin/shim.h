#ifndef ZWS_HIK_SHIM_H
#define ZWS_HIK_SHIM_H

/* Minimal C shim so Go only deals with primitive types, not SDK structs. */

int  hik_init(void);
void hik_cleanup(void);
long hik_last_error(void);

/* Returns user id (>=0) or -1 on failure. */
long hik_login(const char *ip, unsigned short port, const char *user, const char *password);
void hik_logout(long user_id);

/* PTZ: cmd is a NET_DVR_PTZ_* command, stop=1 stops the movement. */
int  hik_ptz(long user_id, long channel, unsigned int cmd, unsigned int stop);

#endif
