//go:build hik && cgo

package main

/*
#cgo CFLAGS: -I${SRCDIR}/sdk/include
#cgo linux LDFLAGS: -L${SRCDIR}/sdk/lib -lhnetsdk -lhpr -lHCCore -lstdc++ -lm
#cgo windows LDFLAGS: -L${SRCDIR}/sdk/lib -lHCNetSDK -lHCCore
#include <stdlib.h>
#include "shim.h"
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// HikSDK 是 HCNetSDK 的最小 cgo 封装；只暴露登录/登出/PTZ。
type HikSDK struct{}

func (HikSDK) Init() error {
	if C.hik_init() != 0 {
		return fmt.Errorf("NET_DVR_Init failed: %d", int(C.hik_last_error()))
	}
	return nil
}

func (HikSDK) Cleanup() { C.hik_cleanup() }

func (HikSDK) Login(ip string, port int, user, password string) (int64, error) {
	cIP := C.CString(ip)
	cUser := C.CString(user)
	cPass := C.CString(password)
	defer C.free(unsafe.Pointer(cIP))
	defer C.free(unsafe.Pointer(cUser))
	defer C.free(unsafe.Pointer(cPass))

	uid := int64(C.hik_login(cIP, C.ushort(port), cUser, cPass))
	if uid < 0 {
		return 0, fmt.Errorf("NET_DVR_Login_V40 failed: %d", int(C.hik_last_error()))
	}
	return uid, nil
}

func (HikSDK) Logout(userID int64) { C.hik_logout(C.long(userID)) }

func (HikSDK) PTZ(userID int64, channel int, cmd int, stop bool) error {
	stopped := 0
	if stop {
		stopped = 1
	}
	if C.hik_ptz(C.long(userID), C.long(channel), C.uint(cmd), C.uint(stopped)) != 0 {
		return fmt.Errorf("NET_DVR_PTZControl_Other failed: %d", int(C.hik_last_error()))
	}
	return nil
}
