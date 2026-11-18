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

#ifdef __cplusplus
}
#endif

#endif
