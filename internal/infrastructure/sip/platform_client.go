package sipinfra

import (
	domainplatform "zero-web-kit/internal/domain/platform"
	"zero-web-kit/internal/infrastructure/config"

	"github.com/zero-pipe/gb28181-go/cascade"
	"github.com/zero-pipe/gb28181-go/manscdp"
)

type PlatformClient struct {
	inner *cascade.Client
	cfg   config.SIPConfig
}

func NewPlatformClient(cfg config.SIPConfig) *PlatformClient {
	return &PlatformClient{
		inner: cascade.NewClient(cascade.Config{ID: cfg.ID, Domain: cfg.Domain}),
		cfg:   cfg,
	}
}

func (c *PlatformClient) ApplyConfig(cfg config.SIPConfig) {
	c.cfg = cfg
	c.inner.ApplyConfig(cascade.Config{ID: cfg.ID, Domain: cfg.Domain})
}

func toUpstream(p *domainplatform.Platform) cascade.Upstream {
	return cascade.Upstream{
		ServerGBID: p.ServerGBID,
		ServerIP:   p.ServerIP,
		ServerPort: p.ServerPort,
		DeviceGBID: p.DeviceGBID,
		Expires:    p.Expires,
	}
}

func (c *PlatformClient) Register(p *domainplatform.Platform) error {
	return c.inner.Register(toUpstream(p))
}

func (c *PlatformClient) Keepalive(p *domainplatform.Platform) error {
	return c.inner.Keepalive(toUpstream(p))
}

func (c *PlatformClient) SendCatalogNotify(p *domainplatform.Platform, items []CatalogItem) error {
	return c.inner.SendCatalogNotify(toUpstream(p), []manscdp.CatalogItem(items))
}

func (c *PlatformClient) Unregister(p *domainplatform.Platform) error {
	return c.inner.Unregister(toUpstream(p))
}
