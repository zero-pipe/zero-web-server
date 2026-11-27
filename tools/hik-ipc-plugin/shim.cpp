//go:build hik && cgo

#include "shim.h"
#include <string.h>
#include <stdlib.h>
#include <stdio.h>
#include <time.h>
#include <stdint.h>

/* Socket 头必须在 HCNetSDK.h(会带上 windows.h) 之前包含，避免 winsock 冲突。 */
#if defined(_WIN32)
#include <winsock2.h>
#include <ws2tcpip.h>
#else
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <unistd.h>
#include <netdb.h>
#endif

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

/* ==========================================================================
 * 实时预览 SDK 取流 → RTP/PS 推送
 *
 * 数据通路：NET_DVR_RealPlay_V40(私有协议) → 回调给 PS → 按 PS pack 头
 * (00 00 01 BA) 分帧 → 每个 pack 作为一组 RTP 包（首包载荷从 BA 起始）→
 * UDP 发到 ZMS openRtpServer 的端口。
 *
 * 与 ZMS 收流器的对齐（zmedia-server src/protocol/rtp/rtp-ps-unpack.c）：
 *   - ZMS 忽略 RTP 的 M 位，改用「RTP 载荷是否以 00 00 01 BA 开头」判定新帧。
 *     所以每组 RTP 包的首包载荷必须正好从 PS pack 头开始，这是正确性关键。
 *   - ZMS 用 seq 做乱序/丢包判定，seq 必须逐包递增。
 *   - PT 默认 96（ZMS_RTP_PS_DEFAULT_PT），SSRC 不过滤，RTP 时间戳被忽略
 *     （PS 的 PES 自带 PTS），这里仍填合理值以兼容其他收流端。
 * ========================================================================== */

#define ZWS_RTP_PS_PT     96
#define ZWS_RTP_MAX_PAYLOAD 1400      /* UDP 载荷上限，留足 MTU 余量 */
#define ZWS_PS_ACC_INIT   (256 * 1024)
#define ZWS_RP_MAX        64          /* 最大并发取流路数 */

#if defined(_WIN32)
#define ZWS_CLOSE_SOCK(s) closesocket(s)
#else
#define ZWS_CLOSE_SOCK(s) close(s)
#endif

typedef struct zws_realplay_ctx {
    LONG real_handle;
    int sock;
    struct sockaddr_in dst;
    uint32_t ssrc;
    uint16_t seq;
    uint32_t ts;
    uint8_t *acc;      /* PS 累积缓冲 */
    size_t acc_len;
    size_t acc_cap;
    volatile int stopped;
    int logged_first;  /* 首次收到码流时打印一次诊断信息 */
} zws_realplay_ctx;

static zws_realplay_ctx *g_rp[ZWS_RP_MAX];

static int zws_acc_reserve(zws_realplay_ctx *c, size_t need) {
    size_t cap;
    uint8_t *p;
    if (c->acc_len + need <= c->acc_cap) return 0;
    cap = c->acc_cap ? c->acc_cap : ZWS_PS_ACC_INIT;
    while (cap < c->acc_len + need) cap *= 2;
    p = (uint8_t *)realloc(c->acc, cap);
    if (!p) return -1;
    c->acc = p;
    c->acc_cap = cap;
    return 0;
}

/* 在 [from, len) 内找下一个 PS pack 头 00 00 01 BA，返回下标或 -1。 */
static long zws_find_ps_pack(const uint8_t *b, size_t len, size_t from) {
    size_t i = from;
    if (len < 4) return -1;
    for (; i + 4 <= len; i++) {
        if (b[i] == 0x00 && b[i + 1] == 0x00 && b[i + 2] == 0x01 && b[i + 3] == 0xBA) {
            return (long)i;
        }
    }
    return -1;
}

/* 把一段完整 PS（以 BA 起始）打成 RTP 包发出去。组内首包载荷从 BA 开始。 */
static void zws_send_rtp_frame(zws_realplay_ctx *c, const uint8_t *ps, size_t len) {
    uint8_t pkt[12 + ZWS_RTP_MAX_PAYLOAD];
    size_t off = 0;

    if (!c || c->sock < 0 || len == 0) return;

    while (off < len) {
        size_t chunk = len - off;
        int last;
        if (chunk > ZWS_RTP_MAX_PAYLOAD) chunk = ZWS_RTP_MAX_PAYLOAD;
        last = (off + chunk >= len);

        pkt[0] = 0x80; /* V=2,P=0,X=0,CC=0 */
        pkt[1] = (uint8_t)((last ? 0x80 : 0x00) | ZWS_RTP_PS_PT); /* M|PT */
        pkt[2] = (uint8_t)(c->seq >> 8);
        pkt[3] = (uint8_t)(c->seq & 0xff);
        pkt[4] = (uint8_t)(c->ts >> 24);
        pkt[5] = (uint8_t)(c->ts >> 16);
        pkt[6] = (uint8_t)(c->ts >> 8);
        pkt[7] = (uint8_t)(c->ts & 0xff);
        pkt[8] = (uint8_t)(c->ssrc >> 24);
        pkt[9] = (uint8_t)(c->ssrc >> 16);
        pkt[10] = (uint8_t)(c->ssrc >> 8);
        pkt[11] = (uint8_t)(c->ssrc & 0xff);
        memcpy(pkt + 12, ps + off, chunk);

        sendto(c->sock, (const char *)pkt, (int)(12 + chunk), 0,
               (struct sockaddr *)&c->dst, sizeof(c->dst));

        c->seq++;
        off += chunk;
    }
    c->ts += 3600; /* 约 40ms@90kHz；ZMS 忽略此值，仅供其他端参考 */
}

/* 追加一段 SDK 回调数据到累积缓冲，并把其中完整的 PS pack 逐份发出去。 */
static void zws_ps_feed(zws_realplay_ctx *c, const uint8_t *data, size_t len) {
    long first, next;
    size_t start;

    if (!c || len == 0) return;
    if (zws_acc_reserve(c, len) != 0) return;
    memcpy(c->acc + c->acc_len, data, len);
    c->acc_len += len;

    /* 丢弃首个 PS pack 头之前的杂字节（SYSHEAD 残留等），只保留尾部 3 字节
     * 以防 pack 头跨回调块被切断。 */
    first = zws_find_ps_pack(c->acc, c->acc_len, 0);
    if (first < 0) {
        if (c->acc_len > 3) {
            memmove(c->acc, c->acc + (c->acc_len - 3), 3);
            c->acc_len = 3;
        }
        return;
    }
    if (first > 0) {
        memmove(c->acc, c->acc + first, c->acc_len - (size_t)first);
        c->acc_len -= (size_t)first;
    }

    /* acc[0] 现在是 BA。以"下一个 BA"为界逐份发出完整 pack： */
    start = 0;
    for (;;) {
        next = zws_find_ps_pack(c->acc, c->acc_len, start + 4);
        if (next < 0) break;
        zws_send_rtp_frame(c, c->acc + start, (size_t)next - start);
        start = (size_t)next;
    }
    if (start > 0) {
        memmove(c->acc, c->acc + start, c->acc_len - start);
        c->acc_len -= start;
    }
}

static void CALLBACK zws_realplay_cb(LONG lRealHandle, DWORD dwDataType, BYTE *pBuffer,
                                     DWORD dwBufSize, void *pUser) {
    zws_realplay_ctx *c = (zws_realplay_ctx *)pUser;
    (void)lRealHandle;
    if (!c || c->stopped || !pBuffer || dwBufSize == 0) return;
    /* 私有协议取流：STREAMDATA 即 PS（音视频复用在同一 PS 流里）。
     * SYSHEAD/AUDIOSTREAMDATA 不参与 RTP/PS，忽略即可。 */
    if (dwDataType == NET_DVR_STREAMDATA) {
        if (!c->logged_first && dwBufSize >= 4) {
            const unsigned char *b = (const unsigned char *)pBuffer;
            int isPS = (b[0] == 0 && b[1] == 0 && b[2] == 1 && b[3] == 0xBA);
            fprintf(stderr,
                    "[hik-rtp] first STREAMDATA len=%u head=%02x%02x%02x%02x ps_header=%d\n",
                    (unsigned)dwBufSize, b[0], b[1], b[2], b[3], isPS);
            c->logged_first = 1;
        }
        zws_ps_feed(c, pBuffer, (size_t)dwBufSize);
    }
}

#if defined(_WIN32)
static int zws_wsa_ready = 0;
#endif

long long hik_realplay_push_start(long user_id, int channel, int stream_type, int link_mode,
                                  const char *dst_ip, unsigned short dst_port, int mode) {
    zws_realplay_ctx *c;
    int slot = -1;
    int i;
    int sndbuf = 4 * 1024 * 1024;
    NET_DVR_PREVIEWINFO pi;
    LONG h;

    (void)mode; /* 目前只支持 UDP 推送（对应 ZMS tcp_mode=0） */

    if (!dst_ip || !dst_ip[0] || dst_port == 0) return -1;
    for (i = 0; i < ZWS_RP_MAX; i++) {
        if (!g_rp[i]) { slot = i; break; }
    }
    if (slot < 0) return -1;

#if defined(_WIN32)
    if (!zws_wsa_ready) {
        WSADATA wsa;
        WSAStartup(MAKEWORD(2, 2), &wsa);
        zws_wsa_ready = 1;
    }
#endif

    c = (zws_realplay_ctx *)calloc(1, sizeof(*c));
    if (!c) return -1;
    c->sock = -1;
    c->real_handle = -1;
    c->ssrc = (uint32_t)(0x5a4d0000u ^ ((uint32_t)channel << 8) ^ (uint32_t)time(NULL));
    c->seq = (uint16_t)(rand() & 0xffff);
    c->ts = 0;

    c->sock = (int)socket(AF_INET, SOCK_DGRAM, 0);
    if (c->sock < 0) {
        free(c);
        return -1;
    }
    setsockopt(c->sock, SOL_SOCKET, SO_SNDBUF, (const char *)&sndbuf, sizeof(sndbuf));

    memset(&c->dst, 0, sizeof(c->dst));
    c->dst.sin_family = AF_INET;
    c->dst.sin_port = htons(dst_port);
    c->dst.sin_addr.s_addr = inet_addr(dst_ip);
    if (c->dst.sin_addr.s_addr == INADDR_NONE) {
        struct hostent *he = gethostbyname(dst_ip);
        if (!he || he->h_addrtype != AF_INET || he->h_length <= 0) {
            ZWS_CLOSE_SOCK(c->sock);
            free(c);
            return -1;
        }
        memcpy(&c->dst.sin_addr, he->h_addr, (size_t)he->h_length);
    }

    memset(&pi, 0, sizeof(pi));
    pi.lChannel = (LONG)channel;
    pi.dwStreamType = (DWORD)(stream_type < 0 ? 0 : stream_type);
    pi.dwLinkMode = (DWORD)(link_mode == 1 ? 1 : 0); /* 0-TCP,1-UDP */
    pi.hPlayWnd = NULL;
    pi.bBlocked = 0;
    pi.byProtoType = 0; /* 0-私有协议：拿到的是 PS 流，且走私有端口不碰 RTSP */
    pi.byReconnect = 1; /* 断线由 SDK 内部重连 */

    h = NET_DVR_RealPlay_V40((LONG)user_id, &pi, zws_realplay_cb, c);
    if (h < 0) {
        int err = (int)NET_DVR_GetLastError();
        ZWS_CLOSE_SOCK(c->sock);
        free(c);
        /* 把 SDK 错误码回填，供 Go 侧 hik_last_error() 读取。 */
        return err > 0 ? -(long long)err : -1;
    }
    c->real_handle = h;
    g_rp[slot] = c;
    return (long long)(slot + 1);
}

void hik_realplay_push_stop(long long handle) {
    zws_realplay_ctx *c;
    int slot;
    if (handle <= 0) return;
    slot = (int)(handle - 1);
    if (slot < 0 || slot >= ZWS_RP_MAX) return;
    c = g_rp[slot];
    if (!c) return;

    c->stopped = 1;
    /* StopRealPlay 返回后该句柄不会再触发回调，之后再释放 ctx 是安全的。 */
    if (c->real_handle >= 0) {
        NET_DVR_StopRealPlay(c->real_handle);
        c->real_handle = -1;
    }
    if (c->sock >= 0) {
        ZWS_CLOSE_SOCK(c->sock);
        c->sock = -1;
    }
    free(c->acc);
    free(c);
    g_rp[slot] = NULL;
}
