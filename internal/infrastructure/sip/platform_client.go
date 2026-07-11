package sipinfra

import (
	"strconv"

	domainplatform "zero-web-kit/internal/domain/platform"
	"zero-web-kit/internal/infrastructure/config"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

type PlatformClient struct {
	cfg    config.SIPConfig
	ua     *sipgo.UserAgent
	client *sipgo.Client
	sn     int
}

func NewPlatformClient(cfg config.SIPConfig) *PlatformClient {
	ua, _ := sipgo.NewUA(sipgo.WithUserAgent(cfg.ID))
	client, _ := sipgo.NewClient(ua)
	return &PlatformClient{cfg: cfg, ua: ua, client: client}
}

func (c *PlatformClient) ApplyConfig(cfg config.SIPConfig) {
	c.cfg = cfg
}

func (c *PlatformClient) Register(p *domainplatform.Platform) error {
	expires := 3600
	if p.Expires != "" {
		if v, err := strconv.Atoi(p.Expires); err == nil {
			expires = v
		}
	}
	recipient := sip.Uri{User: p.ServerGBID, Host: p.ServerIP, Port: p.ServerPort}
	req := sip.NewRequest(sip.REGISTER, recipient)
	req.AppendHeader(sip.NewHeader("Expires", strconv.Itoa(expires)))
	from := &sip.FromHeader{Address: sip.Uri{User: p.DeviceGBID, Host: c.cfg.Domain}}
	to := &sip.FromHeader{Address: recipient}
	req.AppendHeader(from)
	req.AppendHeader(to)
	return c.client.WriteRequest(req)
}

func (c *PlatformClient) Keepalive(p *domainplatform.Platform) error {
	c.sn++
	body := BuildPlatformKeepalive(p.DeviceGBID, strconv.Itoa(c.sn))
	recipient := sip.Uri{User: p.ServerGBID, Host: p.ServerIP, Port: p.ServerPort}
	req := sip.NewRequest(sip.MESSAGE, recipient)
	req.SetBody([]byte(body))
	req.AppendHeader(sip.NewHeader("Content-Type", "Application/MANSCDP+xml"))
	from := &sip.FromHeader{Address: sip.Uri{User: p.DeviceGBID, Host: c.cfg.Domain}}
	to := &sip.FromHeader{Address: recipient}
	req.AppendHeader(from)
	req.AppendHeader(to)
	return c.client.WriteRequest(req)
}

func (c *PlatformClient) SendCatalogNotify(p *domainplatform.Platform, items []CatalogItem) error {
	c.sn++
	body := BuildCatalogNotify(p.DeviceGBID, strconv.Itoa(c.sn), items)
	recipient := sip.Uri{User: p.ServerGBID, Host: p.ServerIP, Port: p.ServerPort}
	req := sip.NewRequest(sip.MESSAGE, recipient)
	req.SetBody([]byte(body))
	req.AppendHeader(sip.NewHeader("Content-Type", "Application/MANSCDP+xml"))
	from := &sip.FromHeader{Address: sip.Uri{User: p.DeviceGBID, Host: c.cfg.Domain}}
	to := &sip.FromHeader{Address: recipient}
	req.AppendHeader(from)
	req.AppendHeader(to)
	return c.client.WriteRequest(req)
}

func (c *PlatformClient) Unregister(p *domainplatform.Platform) error {
	recipient := sip.Uri{User: p.ServerGBID, Host: p.ServerIP, Port: p.ServerPort}
	req := sip.NewRequest(sip.REGISTER, recipient)
	req.AppendHeader(sip.NewHeader("Expires", "0"))
	from := &sip.FromHeader{Address: sip.Uri{User: p.DeviceGBID, Host: c.cfg.Domain}}
	to := &sip.FromHeader{Address: recipient}
	req.AppendHeader(from)
	req.AppendHeader(to)
	return c.client.WriteRequest(req)
}
