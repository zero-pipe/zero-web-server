//go:build hik && cgo

#include "shim.h"
#include <string.h>

/* HCNetSDK headers/libs must be placed under ./sdk (see README). */
#include "HCNetSDK.h"

int hik_init(void) {
    if (!NET_DVR_Init()) {
        return -1;
    }
    NET_DVR_SetConnectTime(3000, 2);
    NET_DVR_SetReconnect(5000, 1);
    return 0;
}

void hik_cleanup(void) {
    NET_DVR_Cleanup();
}

long hik_last_error(void) {
    return (long)NET_DVR_GetLastError();
}

long hik_login(const char *ip, unsigned short port, const char *user, const char *password) {
    NET_DVR_USER_LOGIN_INFO loginInfo;
    NET_DVR_DEVICEINFO_V40 deviceInfo;
    memset(&loginInfo, 0, sizeof(loginInfo));
    memset(&deviceInfo, 0, sizeof(deviceInfo));

    strncpy(loginInfo.sDeviceAddress, ip, sizeof(loginInfo.sDeviceAddress) - 1);
    strncpy(loginInfo.sUserName, user, sizeof(loginInfo.sUserName) - 1);
    strncpy(loginInfo.sPassword, password, sizeof(loginInfo.sPassword) - 1);
    loginInfo.wPort = port;
    loginInfo.bUseAsynLogin = 0;

    LONG uid = NET_DVR_Login_V40(&loginInfo, &deviceInfo);
    return (long)uid;
}

void hik_logout(long user_id) {
    NET_DVR_Logout((LONG)user_id);
}

int hik_ptz(long user_id, long channel, unsigned int cmd, unsigned int stop) {
    if (!NET_DVR_PTZControl_Other((LONG)user_id, (LONG)channel, (DWORD)cmd, (DWORD)stop)) {
        return -1;
    }
    return 0;
}
