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
