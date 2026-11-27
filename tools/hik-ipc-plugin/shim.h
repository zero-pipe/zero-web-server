#ifndef ZWS_HIK_SHIM_H
#define ZWS_HIK_SHIM_H

/* Minimal C shim so Go only deals with primitive types, not SDK structs. */

#ifdef __cplusplus
extern "C" {
#endif

int  hik_init(void);
void hik_cleanup(void);
long hik_last_error(void);

/*
 * Login with extended outputs. Returns user id (>=0) or -1 on failure.
 * out_serial: device serial number (may be empty)
 * out_dev_type: wDevType
 * out_analog_chans: analog channel count
 * out_start_chan: first analog channel
 * out_ip_chans: IP (digital) channel count
 */
long hik_login_ex(const char *ip, unsigned short port, const char *user, const char *password,
                  char *out_serial, int serial_len, int *out_dev_type,
                  int *out_analog_chans, int *out_start_chan, int *out_ip_chans);
void hik_logout(long user_id);

/* Read device name/model/serial via NET_DVR_GET_DEVICECFG_V40. Returns 0 on success. */
int hik_device_cfg(long user_id, char *out_name, int name_len,
                   char *out_model, int model_len, char *out_serial, int serial_len);

/*
 * Enumerate channel numbers. Writes up to max channel numbers into out_ids.
 * Returns count (>=0) or -1 on failure.
 */
int hik_list_channels(long user_id, int start_dchan, int *out_ids, int max);

/* PTZ with speed (1..7). stop=1 stops movement. */
int hik_ptz_speed(long user_id, long channel, unsigned int cmd, unsigned int stop, unsigned int speed);

/* Open/close the device alarm channel. Returns alarm handle (>=0) or -1. */
long hik_setup_alarm(long user_id);
int  hik_close_alarm(long alarm_handle);

/*
 * 录像检索：NET_DVR_FindFile_V40 / NET_DVR_FindNextFile_V40。
 *
 * hik_find_begin 建立检索会话，返回句柄（>=0）；<0 表示建会话失败
 * （通道无权限 / 设备不支持检索），此时用 hik_last_error() 取原因。
 *
 * hik_find_next 每次取一条，语义按 SDK 返回码区分，调用方不要把它当错误处理：
 *    1  取到一条，out_* 有效
 *    0  检索结束。设备没开录像就是这里（SDK 的 1001 NOFIND / 1003 NOMOREFILE）
 *    2  设备仍在检索（SDK 的 1002 ISFINDING），稍后重试
 *   -1  真错误
 *
 * 时间参数为 Unix 秒（按设备本地时间解释）。
 * stream_type: 0-同段只返一种(主>子>三) 1-子码流 2-三码流 3-主码流 254-双码流
 */
long hik_find_begin(long user_id, long channel, long long start_ts, long long end_ts, int stream_type);
int  hik_find_next(long find_handle, char *out_name, int name_len,
                   long long *out_start, long long *out_end, unsigned int *out_size);
void hik_find_close(long find_handle);

/*
 * JPEG 抓图：NET_DVR_CaptureJPEGPicture_NEW，直接把图写进调用方缓冲区，不落临时文件。
 *
 * buf/buf_len 由调用方提供；成功时 out_len 写回实际长度。
 * quality: 0-最好 1-较好 2-一般。分辨率由设备当前码流决定（wPicSize=0xff Auto）。
 * 返回 0 成功，-1 失败（用 hik_last_error() 取原因；常见是缓冲区不够）。
 */
int hik_capture_jpeg(long user_id, long channel, int quality, char *buf, int buf_len, int *out_len);

/*
 * 实时预览 SDK 取流 + RTP/PS 推送（GB28181 式）。
 *
 * 用 SDK 私有协议(byProtoType=0)向设备取流，回调里拿到的是 PS 流；
 * 这里把 PS 按 pack 头(00 00 01 BA)分帧、打成 RTP/PS，用 UDP 发到
 * dst_ip:dst_port（ZMS openRtpServer 分配的收流口）。整条路径走设备的
 * 私有协议端口(8000)，完全不碰 RTSP 554，因此不与 onvif 抢 RTSP 会话。
 *
 * stream_type: 0-主码流 1-子码流 ...（对应 NET_DVR_PREVIEWINFO.dwStreamType）
 * link_mode:   0-TCP 1-UDP（设备连接方式；局域网建议 0）
 * 返回 >=0 的句柄（传给 hik_realplay_push_stop）；-1 失败，hik_last_error() 取原因。
 */
long long hik_realplay_push_start(long user_id, int channel, int stream_type, int link_mode,
                                  const char *dst_ip, unsigned short dst_port, int mode);
void hik_realplay_push_stop(long long handle);

#ifdef __cplusplus
}
#endif

#endif
