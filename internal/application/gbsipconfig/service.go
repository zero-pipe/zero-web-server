package gbsipconfig

import (
	"errors"
	"strings"
	"time"

	"zero-web-kit/internal/infrastructure/config"
	"zero-web-kit/internal/infrastructure/persistence"
	"zero-web-kit/internal/infrastructure/persistence/model"
	sipinfra "zero-web-kit/internal/infrastructure/sip"
	applog "zero-web-kit/pkg/log"

	"gorm.io/gorm"
)

var (
	ErrIPRequired       = errors.New("SIP IP 不能为空")
	ErrPortInvalid      = errors.New("SIP 端口无效")
	ErrDomainRequired   = errors.New("SIP 域不能为空")
	ErrDeviceIDRequired = errors.New("国标编号不能为空")
	ErrDeviceIDLength   = errors.New("国标编号须为20位")
	ErrPasswordRequired = errors.New("SIP 密码不能为空")
)

type OnChangeFunc func(cfg config.SIPConfig, requirePreRegister bool, portChanged bool)

type Service struct {
	repo     *persistence.GbSipConfigRepository
	defaults config.SIPConfig
	onChange OnChangeFunc
}

func NewService(repo *persistence.GbSipConfigRepository, defaults config.SIPConfig) *Service {
	return &Service{repo: repo, defaults: defaults}
}

func (s *Service) SetOnChange(fn OnChangeFunc) {
	s.onChange = fn
}

// Bootstrap 启动入口：库有配置则用库；库空则用 yaml 默认值写入库并返回。
func (s *Service) Bootstrap() (config.SIPConfig, error) {
	row, err := s.repo.Get()
	if err == nil {
		return s.repo.ToSIPConfig(row), nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return config.SIPConfig{}, err
	}
	seed := s.buildDefaultRow()
	if err := s.repo.Save(seed); err != nil {
		return config.SIPConfig{}, err
	}
	cfg := s.repo.ToSIPConfig(seed)
	applog.Info("seeded gb sip config from yaml defaults",
		"id", cfg.ID, "domain", cfg.Domain, "ip", cfg.IP, "port", cfg.Port)
	return cfg, nil
}

func (s *Service) buildDefaultRow() *model.GbSipConfig {
	d := s.defaults
	if d.Port <= 0 {
		d.Port = 5060
	}
	if strings.TrimSpace(d.Domain) == "" {
		d.Domain = "3402000000"
	}
	if strings.TrimSpace(d.ID) == "" {
		d.ID = "34020000002000000001"
	}
	if strings.TrimSpace(d.Password) == "" {
		d.Password = "12345678"
	}
	ip := strings.TrimSpace(d.IP)
	if ip == "" || ip == "0.0.0.0" {
		ip = sipinfra.GuessLocalIP()
	}
	if ip == "" {
		ip = "127.0.0.1"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	return &model.GbSipConfig{
		ID:                 1,
		IP:                 ip,
		Port:               d.Port,
		Domain:             strings.TrimSpace(d.Domain),
		DeviceID:           strings.TrimSpace(d.ID),
		Password:           strings.TrimSpace(d.Password),
		Alarm:              d.Alarm,
		RequirePreRegister: true, // yaml 默认通常为 true；库种子与之一致
		Transport:          "UDP",
		CreateTime:         now,
		UpdateTime:         now,
	}
}

// Load 从库读取；无记录返回空配置。
func (s *Service) Load() (config.SIPConfig, error) {
	row, err := s.repo.Get()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return config.SIPConfig{}, nil
		}
		return config.SIPConfig{}, err
	}
	return s.repo.ToSIPConfig(row), nil
}

// GetOrEmpty 供页面展示；无记录时返回 yaml 默认表单（一般启动已 Bootstrap）。
func (s *Service) GetOrEmpty() (*model.GbSipConfig, error) {
	row, err := s.repo.Get()
	if err == nil {
		return row, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.buildDefaultRow(), nil
	}
	return nil, err
}

func (s *Service) CurrentSIP() (config.SIPConfig, error) {
	return s.Load()
}

func (s *Service) Save(row *model.GbSipConfig) (portChanged bool, err error) {
	if err := validate(row); err != nil {
		return false, err
	}
	oldPort := 0
	if old, getErr := s.repo.Get(); getErr == nil {
		oldPort = old.Port
	}
	if err := s.repo.Save(row); err != nil {
		return false, err
	}
	saved, err := s.repo.Get()
	if err != nil {
		return false, err
	}
	portChanged = oldPort != saved.Port
	if oldPort == 0 && saved.Port > 0 {
		portChanged = true
	}
	cfg := s.repo.ToSIPConfig(saved)
	if s.onChange != nil {
		s.onChange(cfg, saved.RequirePreRegister, portChanged)
	}
	return portChanged, nil
}

func validate(row *model.GbSipConfig) error {
	if row == nil {
		return errors.New("参数错误")
	}
	row.IP = strings.TrimSpace(row.IP)
	row.Domain = strings.TrimSpace(row.Domain)
	row.DeviceID = strings.TrimSpace(row.DeviceID)
	row.Password = strings.TrimSpace(row.Password)
	row.Transport = strings.ToUpper(strings.TrimSpace(row.Transport))
	if row.Transport == "" {
		row.Transport = "UDP"
	}
	if row.Transport != "UDP" && row.Transport != "TCP" {
		return errors.New("传输模式须为 UDP 或 TCP")
	}
	if row.IP == "" {
		return ErrIPRequired
	}
	if row.Port <= 0 || row.Port > 65535 {
		return ErrPortInvalid
	}
	if row.Domain == "" {
		return ErrDomainRequired
	}
	if row.DeviceID == "" {
		return ErrDeviceIDRequired
	}
	if len(row.DeviceID) != 20 {
		return ErrDeviceIDLength
	}
	if row.Password == "" {
		return ErrPasswordRequired
	}
	return nil
}
