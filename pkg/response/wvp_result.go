package response

const (
	CodeSuccess = 0
	CodeError   = 100
	CodeBadReq  = 400
	CodeUnauth  = 401
)

type WVPResult[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

func Success[T any](data T) WVPResult[T] {
	return WVPResult[T]{Code: CodeSuccess, Msg: "success", Data: data}
}

func SuccessMsg[T any](data T, msg string) WVPResult[T] {
	return WVPResult[T]{Code: CodeSuccess, Msg: msg, Data: data}
}

func Fail(code int, msg string) WVPResult[any] {
	return WVPResult[any]{Code: code, Msg: msg, Data: nil}
}

func OK(c interface{ JSON(int, any) }, data any) {
	c.JSON(200, Success(data))
}

func Error(c interface{ JSON(int, any) }, code int, msg string) {
	c.JSON(200, Fail(code, msg))
}
