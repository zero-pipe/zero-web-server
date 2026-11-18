//go:build hik && cgo

#include "shim.h"
#include <string.h>
#include <stdlib.h>
#include <stdio.h>

/* HCNetSDK headers/libs must be placed under ./sdk (see README). */
#include "HCNetSDK.h"

int hik_init(void) {
#if defined(__linux__)
    /* Optional: HIK_SDK_LIB_PATH points to the directory containing
     * HCNetSDKCom/, libcrypto.so.3 and libssl.so.3. */
    const char *libPath = getenv("HIK_SDK_LIB_PATH");
    if (libPath != NULL && libPath[0] != '\0') {
        NET_DVR_LOCAL_SDK_PATH comPath;
        memset(&comPath, 0, sizeof(comPath));
        strncpy(comPath.sPath, libPath, sizeof(comPath.sPath) - 1);
        NET_DVR_SetSDKInitCfg(NET_SDK_INIT_CFG_SDK_PATH, &comPath);

        char buf[2048];
        snprintf(buf, sizeof(buf), "%s/libcrypto.so.3", libPath);
        NET_DVR_SetSDKInitCfg(NET_SDK_INIT_CFG_LIBEAY_PATH, buf);
        snprintf(buf, sizeof(buf), "%s/libssl.so.3", libPath);
        NET_DVR_SetSDKInitCfg(NET_SDK_INIT_CFG_SSLEAY_PATH, buf);
    }
#endif
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

static void copy_str(char *dst, int dst_len, const char *src, int src_len) {
    if (dst == NULL || dst_len <= 0) return;
    int n = src_len < dst_len - 1 ? src_len : dst_len - 1;
    if (n < 0) n = 0;
    if (n > 0 && src != NULL) {
        memcpy(dst, src, (size_t)n);
    }
    dst[n] = '\0';
    /* trim trailing NULs/spaces */
    for (int i = n - 1; i >= 0 && (dst[i] == '\0' || dst[i] == ' '); i--) {
        dst[i] = '\0';
    }
}

long hik_login_ex(const char *ip, unsigned short port, const char *user, const char *password,
                  char *out_serial, int serial_len, int *out_dev_type,
                  int *out_analog_chans, int *out_start_chan, int *out_ip_chans) {
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
    if (uid < 0) {
        return -1;
    }

    NET_DVR_DEVICEINFO_V30 *v30 = &deviceInfo.struDeviceV30;
    copy_str(out_serial, serial_len, (const char *)v30->sSerialNumber, SERIALNO_LEN);
    if (out_dev_type) *out_dev_type = (int)v30->wDevType;
    if (out_analog_chans) *out_analog_chans = (int)v30->byChanNum;
    if (out_start_chan) *out_start_chan = (int)v30->byStartChan;
    if (out_ip_chans) {
        int ipChans = (int)v30->byIPChanNum + ((int)v30->byHighDChanNum << 8);
        *out_ip_chans = ipChans;
    }
    return (long)uid;
}

void hik_logout(long user_id) {
    NET_DVR_Logout((LONG)user_id);
}

int hik_device_cfg(long user_id, char *out_name, int name_len,
                   char *out_model, int model_len, char *out_serial, int serial_len) {
    NET_DVR_DEVICECFG_V40 cfg;
    DWORD returned = 0;
    memset(&cfg, 0, sizeof(cfg));
    cfg.dwSize = sizeof(cfg);
    if (!NET_DVR_GetDVRConfig((LONG)user_id, NET_DVR_GET_DEVICECFG_V40, 0xffffffff,
                              &cfg, sizeof(cfg), &returned)) {
        return -1;
    }
    copy_str(out_name, name_len, (const char *)cfg.sDVRName, NAME_LEN);
    copy_str(out_model, model_len, (const char *)cfg.byDevTypeName, DEV_TYPE_NAME_LEN);
    copy_str(out_serial, serial_len, (const char *)cfg.sSerialNumber, SERIALNO_LEN);
    return 0;
}

int hik_list_channels(long user_id, int start_dchan, int *out_ids, int max) {
    NET_DVR_IPPARACFG_V40 cfg;
    DWORD returned = 0;
    memset(&cfg, 0, sizeof(cfg));
    cfg.dwSize = sizeof(cfg);

    BOOL ok = NET_DVR_GetDVRConfig((LONG)user_id, NET_DVR_GET_IPPARACFG_V40, 1,
                                   &cfg, sizeof(cfg), &returned);
    if (!ok) {
        ok = NET_DVR_GetDVRConfig((LONG)user_id, NET_DVR_GET_IPPARACFG_V40, 0xffffffff,
                                  &cfg, sizeof(cfg), &returned);
    }

    int count = 0;
    if (ok) {
        int start = (int)cfg.dwStartDChan;
        if (start <= 0) start = start_dchan;
        if (start <= 0) start = 1;
        for (int i = 0; i < MAX_CHANNUM_V30 && count < max; i++) {
            BYTE enabled = cfg.struStreamMode[i].uGetStream.struIPChan.byEnable;
            if (!enabled) continue;
            int ch = start + i;
            if (ch <= 0) ch = i + 1;
            out_ids[count++] = ch;
        }
        /* analog channels */
        if (count == 0) {
            int analog = (int)cfg.dwAChanNum;
            for (int i = 0; i < analog && i < MAX_CHANNUM_V30 && count < max; i++) {
                if (cfg.byAnalogChanEnable[i]) {
                    out_ids[count++] = i + 1;
                }
            }
        }
        if (count > 0) return count;
    }

    /* Fallback: 1..N (direct IPC / unsupported config). */
    int n = start_dchan > 0 ? start_dchan : 1;
    for (int i = 0; i < n && count < max; i++) {
        out_ids[count++] = i + 1;
    }
    return count;
}

int hik_ptz_speed(long user_id, long channel, unsigned int cmd, unsigned int stop, unsigned int speed) {
    if (speed < 1) speed = 1;
    if (speed > 7) speed = 7;
    if (!NET_DVR_PTZControlWithSpeed_Other((LONG)user_id, (LONG)channel,
                                           (DWORD)cmd, (DWORD)stop, (DWORD)speed)) {
        return -1;
    }
    return 0;
}
