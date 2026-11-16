package port

import (
	domainplatform "zero-web-server/internal/domain/platform"
	"zero-web-server/internal/gbsip"
)

// PlatformClient 是向级联上级平台注册/保活/推送目录的信令端口。
type PlatformClient interface {
	Register(platform *domainplatform.Platform) error
	Keepalive(platform *domainplatform.Platform) error
	SendCatalogNotify(platform *domainplatform.Platform, items []gbsip.CatalogItem) error
	Unregister(platform *domainplatform.Platform) error
}
