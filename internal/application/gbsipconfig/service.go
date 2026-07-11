package gbsipconfig

import (
	"errors"
	"strings"

	"zero-web-kit/internal/infrastructure/config"
	"zero-web-kit/internal/infrastructure/persistence"
	"zero-web-kit/internal/infrastructure/persistence/model"

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

type OnChangeFunc func(cfg config.SIPConfig, portChanged bool)

type Service struct {
	repo     *persistence.GbSipConfigRepository
	onChange OnChangeFunc
}

func NewService(repo *persistence.GbSipConfigRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) SetOnChange(fn OnChangeFunc) {
	s.onChange = fn
}

// Load 从库读取；无记录返回空配置（不自动建行，由页面手动保存）。
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

// GetOrEmpty 供页面展示；无记录时返回可编辑的空表单对象。
func (s *Service) GetOrEmpty() (*model.GbSipConfig, error) {
	row, err := s.repo.Get()
	if err == nil {
		return row, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &model.GbSipConfig{ID: 1, Port: 5060}, nil
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
		s.onChange(cfg, portChanged)
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
