package port

import (
	"context"

	onviflib "zero-web-server/3rdpart/onvif-go"
)

// ONVIFFactory 构造 / 初始化 ONVIF 客户端。
// 返回第三方 vendored 库的 *onviflib.Client（第三方类型，不属于本项目基础设施）。
type ONVIFFactory interface {
	NewClient(endpoint, username, password string) (*onviflib.Client, error)
	Initialize(ctx context.Context, endpoint, username, password string) (*onviflib.Client, error)
}
