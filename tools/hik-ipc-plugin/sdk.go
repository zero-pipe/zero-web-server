//go:build hik && cgo

package main

/*
#cgo CFLAGS: -I${SRCDIR}/sdk/include
#cgo CXXFLAGS: -I${SRCDIR}/sdk/include
#cgo linux LDFLAGS: -L${SRCDIR}/sdk/lib -lhnetsdk -lhpr -lHCCore -lstdc++ -lm -lpthread -ldl -Wl,-rpath,$ORIGIN/hik-lib
#cgo windows LDFLAGS: -L${SRCDIR}/sdk/lib -lHCNetSDK -lHCCore -lstdc++ -static-libgcc -static-libstdc++
#include <stdlib.h>
#include "shim.h"
*/
import "C"

import (
	"fmt"
	"time"
	"unsafe"
)

// DeviceInfo is the subset of HCNetSDK login/device config we use.
type DeviceInfo struct {
	Serial      string
	Name        string
	Model       string
	DevType     int
	AnalogChans int
	StartChan   int
	IPChans     int
}

// HikSDK is the cgo wrapper around HCNetSDK (login / device info / channels / PTZ).
type HikSDK struct{}

func (HikSDK) Init() error {
	if C.hik_init() != 0 {
		return fmt.Errorf("NET_DVR_Init failed: %d", int(C.hik_last_error()))
	}
	return nil
}

func (HikSDK) Cleanup() { C.hik_cleanup() }

func (HikSDK) Login(ip string, port int, user, password string) (int64, *DeviceInfo, error) {
	cIP := C.CString(ip)
	cUser := C.CString(user)
	cPass := C.CString(password)
	defer C.free(unsafe.Pointer(cIP))
	defer C.free(unsafe.Pointer(cUser))
	defer C.free(unsafe.Pointer(cPass))

	var serial [256]C.char
	var devType, analogChans, startChan, ipChans C.int
	uid := int64(C.hik_login_ex(
		cIP, C.ushort(port), cUser, cPass,
		&serial[0], C.int(len(serial)),
		&devType, &analogChans, &startChan, &ipChans,
	))
	if uid < 0 {
		return 0, nil, fmt.Errorf("NET_DVR_Login_V40 failed: %d", int(C.hik_last_error()))
	}
	info := &DeviceInfo{
		Serial:      C.GoString(&serial[0]),
		DevType:     int(devType),
		AnalogChans: int(analogChans),
		StartChan:   int(startChan),
		IPChans:     int(ipChans),
	}
	if cfg, err := (HikSDK{}).DeviceCfg(uid); err == nil && cfg != nil {
		if cfg.Name != "" {
			info.Name = cfg.Name
		}
		if cfg.Model != "" {
			info.Model = cfg.Model
		}
		if info.Serial == "" && cfg.Serial != "" {
			info.Serial = cfg.Serial
		}
	}
	return uid, info, nil
}

func (HikSDK) DeviceCfg(userID int64) (*DeviceInfo, error) {
	var name, model, serial [256]C.char
	if C.hik_device_cfg(C.long(userID),
		&name[0], C.int(len(name)),
		&model[0], C.int(len(model)),
		&serial[0], C.int(len(serial))) != 0 {
		return nil, fmt.Errorf("NET_DVR_GET_DEVICECFG_V40 failed: %d", int(C.hik_last_error()))
	}
	return &DeviceInfo{
		Name: C.GoString(&name[0]), Model: C.GoString(&model[0]), Serial: C.GoString(&serial[0]),
	}, nil
}

func (HikSDK) ListChannels(userID int64, startDChan, max int) []int {
	if max <= 0 {
		max = 64
	}
	out := make([]C.int, max)
	n := int(C.hik_list_channels(C.long(userID), C.int(startDChan), &out[0], C.int(max)))
	if n <= 0 {
		return nil
	}
	ids := make([]int, 0, n)
	for i := 0; i < n && i < len(out); i++ {
		ids = append(ids, int(out[i]))
	}
	return ids
}

func (HikSDK) Logout(userID int64) { C.hik_logout(C.long(userID)) }

func (HikSDK) SetupAlarm(userID int64) (int64, error) {
	handle := int64(C.hik_setup_alarm(C.long(userID)))
	if handle < 0 {
		return 0, fmt.Errorf("NET_DVR_SetupAlarmChan_V41 failed: %d", int(C.hik_last_error()))
	}
	return handle, nil
}

func (HikSDK) CloseAlarm(handle int64) error {
	if C.hik_close_alarm(C.long(handle)) != 0 {
		return fmt.Errorf("NET_DVR_CloseAlarmChan_V30 failed: %d", int(C.hik_last_error()))
	}
	return nil
}

func (HikSDK) PTZ(userID int64, channel int, cmd int, stop bool, speed int) error {
	stopped := 0
	if stop {
		stopped = 1
	}
	if C.hik_ptz_speed(C.long(userID), C.long(channel), C.uint(cmd), C.uint(stopped), C.uint(speed)) != 0 {
		return fmt.Errorf("NET_DVR_PTZControlWithSpeed_Other failed: %d", int(C.hik_last_error()))
	}
	return nil
}

// CaptureJPEG 抓一张 JPEG。分辨率由设备当前码流决定。
//
// 缓冲区一次给 4MB：自动分辨率下 1080p/4MP 的图通常 200KB~1.5MB，
// 留足余量避免"缓冲区不够"这类要排查半天的小坑。
func (HikSDK) CaptureJPEG(userID int64, channel int, quality int) ([]byte, error) {
	const maxJPEG = 4 << 20
	buf := make([]byte, maxJPEG)
	var outLen C.int
	if C.hik_capture_jpeg(C.long(userID), C.long(channel), C.int(quality),
		(*C.char)(unsafe.Pointer(&buf[0])), C.int(len(buf)), &outLen) != 0 {
		return nil, fmt.Errorf("NET_DVR_CaptureJPEGPicture_NEW failed: %d", int(C.hik_last_error()))
	}
	n := int(outLen)
	if n <= 0 || n > len(buf) {
		return nil, fmt.Errorf("抓图返回长度异常: %d", n)
	}
	return buf[:n], nil
}

// RecordFile 是一条设备录像。时间为 Unix 秒，按设备本地时间解释。
type RecordFile struct {
	Name      string `json:"name"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
	Size      uint32 `json:"size"`
}

// 录像检索时 SDK 返回"仍在检索"的轮询节奏与整体上限。
const (
	findPollInterval = 200 * time.Millisecond
	findPollTimeout  = 10 * time.Second
)

// FindRecords 检索通道在 [start, end] 内的设备录像。
//
// 设备没开录像时返回空列表且 err == nil：SDK 的 1001(NOFIND)/1003(NOMOREFILE)
// 表示"查不到/取完了"，是正常结束而不是错误，不能当异常抛给上层。
func (HikSDK) FindRecords(userID int64, channel int, start, end int64, streamType int) ([]RecordFile, error) {
	handle := int64(C.hik_find_begin(C.long(userID), C.long(channel),
		C.longlong(start), C.longlong(end), C.int(streamType)))
	if handle < 0 {
		return nil, fmt.Errorf("NET_DVR_FindFile_V40 failed: %d", int(C.hik_last_error()))
	}
	defer C.hik_find_close(C.long(handle))

	var name [100]C.char
	var startTS, endTS C.longlong
	var size C.uint

	out := make([]RecordFile, 0, 8)
	deadline := time.Now().Add(findPollTimeout)
	for {
		switch int(C.hik_find_next(C.long(handle), &name[0], C.int(len(name)),
			&startTS, &endTS, &size)) {
		case 1:
			out = append(out, RecordFile{
				Name:      C.GoString(&name[0]),
				StartTime: int64(startTS),
				EndTime:   int64(endTS),
				Size:      uint32(size),
			})
		case 0:
			return out, nil
		case 2:
			if time.Now().After(deadline) {
				return out, fmt.Errorf("录像检索超时：设备检索时间过长")
			}
			time.Sleep(findPollInterval)
		default:
			return nil, fmt.Errorf("NET_DVR_FindNextFile_V40 failed: %d", int(C.hik_last_error()))
		}
	}
}
