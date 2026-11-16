package response

import (
	"errors"

	"zero-web-server/internal/apperr"
)

const (
	CodeSuccess   = 0
	CodeError     = 100
	CodeBadReq    = 400
	CodeUnauth    = 401
	CodeForbidden = 403
)

type ZWSResult[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

func Success[T any](data T) ZWSResult[T] {
	return ZWSResult[T]{Code: CodeSuccess, Msg: "success", Data: data}
}

func SuccessMsg[T any](data T, msg string) ZWSResult[T] {
	return ZWSResult[T]{Code: CodeSuccess, Msg: msg, Data: data}
}

func Fail(code int, msg string) ZWSResult[any] {
	return ZWSResult[any]{Code: code, Msg: msg, Data: nil}
}

func OK(c interface{ JSON(int, any) }, data any) {
	c.JSON(200, Success(data))
}

func Error(c interface{ JSON(int, any) }, code int, msg string) {
	c.JSON(200, Fail(code, msg))
}

// FromError 将应用错误映射为响应：*apperr.AppError 用其码与消息，其余回落 CodeError。
func FromError(c interface{ JSON(int, any) }, err error) {
	if err == nil {
		OK(c, nil)
		return
	}
	var ae *apperr.AppError
	if errors.As(err, &ae) {
		Error(c, ae.Code, ae.Msg)
		return
	}
	Error(c, CodeError, err.Error())
}

// FromErrorMsg 在无底层 err 时按码段返回统一消息。
func FromErrorMsg(c interface{ JSON(int, any) }, code int, msg string) {
	Error(c, code, msg)
}
