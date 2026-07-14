package subordinateapp

import (
	"errors"
	"strings"
	"time"

	domainsub "zero-web-kit/internal/domain/subordinate"
	gbserver "github.com/zero-pipe/gb28181-go/server"
)

var (
	ErrNameRequired   = errors.New("名称不能为空")
	ErrGBIDRequired   = errors.New("下级平台国标编号不能为空")
	ErrGBIDLength     = errors.New("下级平台国标编号须为20位")
	ErrPasswordRequired = errors.New("注册密码不能为空")
	ErrDisabled       = errors.New("下级平台未启用")
)

type Service struct {
	repo     domainsub.Repository
	serverID string
}

func NewService(repo domainsub.Repository, serverID string) *Service {
	return &Service{repo: repo, serverID: serverID}
}

func (s *Service) List(page, count int, query string) ([]*domainsub.Platform, int64, error) {
	return s.repo.List(page, count, query)
}

func (s *Service) Get(id int) (*domainsub.Platform, error) {
	return s.repo.GetByID(id)
}

func (s *Service) Add(p *domainsub.Platform) error {
	if err := validateSub(p); err != nil {
		return err
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	p.CreateTime = now
	p.UpdateTime = now
	p.ServerID = s.serverID
	p.Status = false
	if p.Transport == "" {
		p.Transport = "UDP"
	}
	return s.repo.Create(p)
}

func (s *Service) Update(p *domainsub.Platform) error {
	if err := validateSub(p); err != nil {
		return err
	}
	p.UpdateTime = time.Now().Format("2006-01-02 15:04:05")
	return s.repo.Update(p)
}

func (s *Service) Delete(id int) error {
	return s.repo.Delete(id)
}

// ResolvePassword implements SIP auth lookup for subordinates.
func (s *Service) ResolvePassword(gbID string) (password string, known bool, err error) {
	p, err := s.repo.GetByGBID(gbID)
	if err != nil || p == nil {
		return "", false, nil
	}
	if !p.Enable {
		// 未启用视为未知，触发 403（强制预登记时）
		return "", false, nil
	}
	return p.Password, true, nil
}

func (s *Service) Known(gbID string) bool {
	p, err := s.repo.GetByGBID(gbID)
	return err == nil && p != nil && p.Enable
}

func (s *Service) Exists(gbID string) bool {
	p, err := s.repo.GetByGBID(gbID)
	return err == nil && p != nil
}

func (s *Service) OnRegister(ev gbserver.RegisterEvent) error {
	p, err := s.repo.GetByGBID(ev.DeviceID)
	if err != nil || p == nil {
		return errors.New("下级平台未预登记")
	}
	if !p.Enable {
		return ErrDisabled
	}
	return s.repo.UpdateOnline(ev.DeviceID, ev.IP, ev.Port, ev.Expires, ev.CallID, ev.Transport)
}

func (s *Service) OnUnregister(gbID string) error {
	return s.repo.UpdateOffline(gbID)
}

func (s *Service) OnKeepalive(gbID, ip string, port int) error {
	p, err := s.repo.GetByGBID(gbID)
	if err != nil || p == nil {
		return nil
	}
	if ip == "" {
		return s.repo.UpdateOnline(gbID, p.IP, p.Port, p.Expires, p.RegisterCall, p.Transport)
	}
	return s.repo.UpdateOnline(gbID, ip, port, p.Expires, p.RegisterCall, p.Transport)
}

func validateSub(p *domainsub.Platform) error {
	if p == nil {
		return errors.New("参数错误")
	}
	p.Name = strings.TrimSpace(p.Name)
	p.DeviceGBID = strings.TrimSpace(p.DeviceGBID)
	p.Password = strings.TrimSpace(p.Password)
	p.Transport = strings.ToUpper(strings.TrimSpace(p.Transport))
	if p.Name == "" {
		return ErrNameRequired
	}
	if p.DeviceGBID == "" {
		return ErrGBIDRequired
	}
	if len(p.DeviceGBID) != 20 {
		return ErrGBIDLength
	}
	if p.Password == "" {
		return ErrPasswordRequired
	}
	if p.Transport == "" {
		p.Transport = "UDP"
	}
	return nil
}
