package response

const (
	CodeSuccess = 0
	CodeError   = 100
	CodeBadReq  = 400
	CodeUnauth  = 401
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
