// Package apperr 定义可携带业务错误码的应用错误。
package apperr

import "fmt"

// 错误码段（兼容旧语义：0/100/400/401/403 保持不变）。
const (
	CodeGeneric = 10000 // 通用业务
	CodeDevice  = 11000 // 设备
	CodeMedia   = 12000 // 媒体节点
	CodeStream  = 13000 // 流 / 代理 / 推流
	CodeStorage = 14000 // 存储 / 云录
	CodeUser    = 15000 // 用户 / 鉴权
	CodeCascade = 16000 // 级联 / 上级平台
	CodeGroup   = 17000 // 分组 / 区域
)

// AppError 携带业务错误码与用户可读消息。
type AppError struct {
	Code int
	Msg  string
	Err  error
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Msg != "" {
		return e.Msg
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("error code %d", e.Code)
}

func (e *AppError) Unwrap() error { return e.Err }

// New 创建带码错误。
func New(code int, msg string) *AppError {
	return &AppError{Code: code, Msg: msg}
}

// Wrap 包装底层错误，保留 call chain。
func Wrap(err error, code int, msg string) *AppError {
	return &AppError{Code: code, Msg: msg, Err: err}
}
