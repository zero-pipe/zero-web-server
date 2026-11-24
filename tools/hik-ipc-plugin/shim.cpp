//go:build hik && cgo

#include "shim.h"
#include <string.h>
#include <stdlib.h>
#include <stdio.h>
#include <time.h>

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

static void CALLBACK zws_alarm_cb(LONG lCommand, NET_DVR_ALARMER *pAlarmer,
                                  char *pAlarmInfo, DWORD dwBufLen, void *pUser) {
    /* Step 3 will parse and publish events. For now the callback only keeps
     * the alarm channel alive. */
    (void)lCommand; (void)pAlarmer; (void)pAlarmInfo; (void)dwBufLen; (void)pUser;
}

long hik_setup_alarm(long user_id) {
    NET_DVR_SetDVRMessageCallBack_V50(0, zws_alarm_cb, NULL);
    NET_DVR_SETUPALARM_PARAM param;
    memset(&param, 0, sizeof(param));
    param.dwSize = sizeof(param);
    param.byLevel = 1;           /* medium */
    param.byAlarmInfoType = 1;   /* new alarm info (V40) */
    param.byRetAlarmTypeV40 = 1;
    param.byDeployType = 0;      /* client deployment */
    LONG handle = NET_DVR_SetupAlarmChan_V41((LONG)user_id, &param);
    return (long)handle;
}

int hik_close_alarm(long alarm_handle) {
    if (alarm_handle < 0) return 0;
    return NET_DVR_CloseAlarmChan_V30((LONG)alarm_handle) ? 0 : -1;
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

/* Unix 秒 <-> NET_DVR_TIME。设备用的是本地时间，两侧都走 localtime/mktime 以保持对称。 */
static void ts_to_dvr(long long ts, NET_DVR_TIME *t) {
    time_t raw = (time_t)ts;
    struct tm tmv;
    memset(&tmv, 0, sizeof(tmv));
#if defined(_WIN32)
    localtime_s(&tmv, &raw);
#else
    localtime_r(&raw, &tmv);
#endif
    t->dwYear   = (DWORD)(tmv.tm_year + 1900);
    t->dwMonth  = (DWORD)(tmv.tm_mon + 1);
    t->dwDay    = (DWORD)tmv.tm_mday;
    t->dwHour   = (DWORD)tmv.tm_hour;
    t->dwMinute = (DWORD)tmv.tm_min;
    t->dwSecond = (DWORD)tmv.tm_sec;
}

static long long dvr_to_ts(const NET_DVR_TIME *t) {
    struct tm tmv;
    memset(&tmv, 0, sizeof(tmv));
    tmv.tm_year  = (int)t->dwYear - 1900;
    tmv.tm_mon   = (int)t->dwMonth - 1;
    tmv.tm_mday  = (int)t->dwDay;
    tmv.tm_hour  = (int)t->dwHour;
    tmv.tm_min   = (int)t->dwMinute;
    tmv.tm_sec   = (int)t->dwSecond;
    tmv.tm_isdst = -1;
    return (long long)mktime(&tmv);
}

long hik_find_begin(long user_id, long channel, long long start_ts, long long end_ts, int stream_type) {
    NET_DVR_FILECOND_V40 cond;
    memset(&cond, 0, sizeof(cond));
    cond.lChannel   = (LONG)channel;
    cond.dwFileType = 0xff; /* 全部录像类型 */
    cond.dwIsLocked = 0xff; /* 锁定与未锁定都查 */
    ts_to_dvr(start_ts, &cond.struStartTime);
    ts_to_dvr(end_ts, &cond.struStopTime);
    cond.byStreamType = (BYTE)(stream_type & 0xff);
    LONG h = NET_DVR_FindFile_V40((LONG)user_id, &cond);
    return (long)h;
}

int hik_find_next(long find_handle, char *out_name, int name_len,
                  long long *out_start, long long *out_end, unsigned int *out_size) {
    NET_DVR_FINDDATA_V40 data;
    memset(&data, 0, sizeof(data));
    int r = (int)NET_DVR_FindNextFile_V40((LONG)find_handle, &data);
    if (r == NET_DVR_FILE_SUCCESS) {
        /* sFileName 是 char[100] */
        copy_str(out_name, name_len, data.sFileName, 100);
        if (out_start) *out_start = dvr_to_ts(&data.struStartTime);
        if (out_end)   *out_end   = dvr_to_ts(&data.struStopTime);
        if (out_size)  *out_size  = (unsigned int)data.dwFileSize;
        return 1;
    }
    if (r == NET_DVR_ISFINDING) return 2;
    /* 1001 NOFIND / 1003 NOMOREFILE：没开录像或已取完，是正常结束，不是错误。 */
    if (r == NET_DVR_FILE_NOFIND || r == NET_DVR_NOMOREFILE) return 0;
    return -1;
}

void hik_find_close(long find_handle) {
    if (find_handle < 0) return;
    NET_DVR_FindClose((LONG)find_handle);
}

int hik_capture_jpeg(long user_id, long channel, int quality, char *buf, int buf_len, int *out_len) {
    if (buf == NULL || buf_len <= 0 || out_len == NULL) return -1;
    NET_DVR_JPEGPARA para;
    memset(&para, 0, sizeof(para));
    para.wPicSize    = 0xff;                 /* Auto：跟随设备当前码流分辨率 */
    para.wPicQuality = (WORD)(quality < 0 ? 1 : (quality > 2 ? 2 : quality));
    DWORD returned = 0;
    if (!NET_DVR_CaptureJPEGPicture_NEW((LONG)user_id, (LONG)channel, &para,
                                        buf, (DWORD)buf_len, &returned)) {
        return -1;
    }
    if (returned == 0) return -1;
    *out_len = (int)returned;
    return 0;
}
