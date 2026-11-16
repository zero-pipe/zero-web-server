package gbsipconfig

import (
	"errors"
	"strings"
	"time"
	"zero-web-server/internal/apperr"

	domaingbsipconfig "zero-web-server/internal/domain/gbsipconfig"
	"zero-web-server/internal/gbsip"
	"zero-web-server/internal/infrastructure/config"
	applog "zero-web-server/pkg/log"
)

var (
	ErrIPRequired       = apperr.New(apperr.CodeGeneric+1, "SIP IP 不能为空")
	ErrPortInvalid      = apperr.New(apperr.CodeGeneric+2, "SIP 端口无效")
	ErrDomainRequired   = apperr.New(apperr.CodeGeneric+3, "SIP 域不能为空")
	ErrDeviceIDRequired = apperr.New(apperr.CodeGeneric+4, "国标编号不能为空")
	ErrDeviceIDLength   = apperr.New(apperr.CodeGeneric+5, "国标编号须为20位")
	ErrPasswordRequired = apperr.New(apperr.CodeGeneric+6, "SIP 密码不能为空")
)

type OnChangeFunc func(cfg config.SIPConfig, requirePreRegister bool, portChanged bool)

type Service struct {
	repo     domaingbsipconfig.Repository
	defaults config.SIPConfig
	onChange OnChangeFunc
}

func NewService(repo domaingbsipconfig.Repository, defaults config.SIPConfig) *Service {
	return &Service{repo: repo, defaults: defaults}
}

func (s *Service) SetOnChange(fn OnChangeFunc) {
	s.onChange = fn
}

// Bootstrap 启动入口：库有配置则用库；库空则用 yaml 默认值写入库并返回。
func (s *Service) Bootstrap() (config.SIPConfig, error) {
	row, err := s.repo.Get()
	if err == nil {
		return toSIPConfig(row), nil
	}
	if !errors.Is(err, domaingbsipconfig.ErrNotFound) {
		return config.SIPConfig{}, err
	}
	seed := s.buildDefaultRow()
	if err := s.repo.Save(seed); err != nil {
		return config.SIPConfig{}, err
	}
	cfg := toSIPConfig(seed)
	applog.Info("seeded gb sip config from yaml defaults",
		"id", cfg.ID, "domain", cfg.Domain, "ip", cfg.IP, "port", cfg.Port)
	return cfg, nil
}

func (s *Service) buildDefaultRow() *domaingbsipconfig.Config {
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
		ip = gbsip.GuessLocalIP()
	}
	if ip == "" {
		ip = "127.0.0.1"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	return &domaingbsipconfig.Config{
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
		if errors.Is(err, domaingbsipconfig.ErrNotFound) {
			return config.SIPConfig{}, nil
		}
		return config.SIPConfig{}, err
	}
	return toSIPConfig(row), nil
}

// GetOrEmpty 供页面展示；无记录时返回 yaml 默认表单（一般启动已 Bootstrap）。
func (s *Service) GetOrEmpty() (*domaingbsipconfig.Config, error) {
	row, err := s.repo.Get()
	if err == nil {
		return row, nil
	}
	if errors.Is(err, domaingbsipconfig.ErrNotFound) {
		return s.buildDefaultRow(), nil
	}
	return nil, err
}

func (s *Service) CurrentSIP() (config.SIPConfig, error) {
	return s.Load()
}

func (s *Service) Save(row *domaingbsipconfig.Config) (portChanged bool, err error) {
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
	cfg := toSIPConfig(saved)
	if s.onChange != nil {
		s.onChange(cfg, saved.RequirePreRegister, portChanged)
	}
	return portChanged, nil
}

func validate(row *domaingbsipconfig.Config) error {
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

func toSIPConfig(row *domaingbsipconfig.Config) config.SIPConfig {
	if row == nil {
		return config.SIPConfig{}
	}
	return config.SIPConfig{
		IP: row.IP, Port: row.Port, Domain: row.Domain,
		ID: row.DeviceID, Password: row.Password, Alarm: row.Alarm,
	}
}
