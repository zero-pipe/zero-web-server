package dictapp

import (
	"strings"

	domaindict "zero-web-server/internal/domain/dict"
)

// Service 数据字典业务逻辑。
type Service struct {
	repo domaindict.Repository
}

func NewService(repo domaindict.Repository) *Service {
	return &Service{repo: repo}
}

type badReqError struct{ msg string }

func (e badReqError) Error() string { return e.msg }
func badReq(msg string) error       { return badReqError{msg} }

// IsBadRequest 是否参数类错误（供 handler 映射 400）。
func IsBadRequest(err error) bool {
	_, ok := err.(badReqError)
	return ok
}

func (s *Service) ListTypes() ([]domaindict.Type, error) {
	return s.repo.ListTypes()
}

// ListItems 按类型取字典项（enabledOnly=true 时仅返回启用项，供下拉）。
func (s *Service) ListItems(typeCode string, enabledOnly bool) ([]domaindict.Item, error) {
	return s.repo.ListItems(strings.TrimSpace(typeCode), enabledOnly)
}

func (s *Service) CreateType(code, name, remark string, sort int) (*domaindict.Type, error) {
	code = strings.TrimSpace(code)
	name = strings.TrimSpace(name)
	if code == "" {
		return nil, badReq("类型码不能为空")
	}
	if name == "" {
		name = code
	}
	if _, err := s.repo.GetTypeByCode(code); err == nil {
		return nil, badReq("类型码已存在: " + code)
	}
	t := &domaindict.Type{Code: code, Name: name, Remark: strings.TrimSpace(remark), Sort: sort}
	if err := s.repo.CreateType(t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) UpdateType(id int64, name, remark string, sort int) error {
	t, err := s.repo.GetTypeByID(id)
	if err != nil {
		return err
	}
	if n := strings.TrimSpace(name); n != "" {
		t.Name = n
	}
	t.Remark = strings.TrimSpace(remark)
	t.Sort = sort
	return s.repo.UpdateType(t)
}

func (s *Service) DeleteType(id int64) error {
	t, err := s.repo.GetTypeByID(id)
	if err != nil {
		return err
	}
	if t.Builtin {
		return badReq("内置类型不可删除")
	}
	// 删类型时连带删除其自定义项
	items, _ := s.repo.ListItems(t.Code, false)
	for i := range items {
		_ = s.repo.DeleteItem(items[i].ID)
	}
	return s.repo.DeleteType(id)
}

func (s *Service) CreateItem(typeCode, value, label string, sort int, enabled bool, remark string) (*domaindict.Item, error) {
	typeCode = strings.TrimSpace(typeCode)
	value = strings.TrimSpace(value)
	label = strings.TrimSpace(label)
	if typeCode == "" || value == "" {
		return nil, badReq("类型和值不能为空")
	}
	if _, err := s.repo.GetTypeByCode(typeCode); err != nil {
		return nil, badReq("类型不存在: " + typeCode)
	}
	if label == "" {
		label = value
	}
	it := &domaindict.Item{
		TypeCode: typeCode, Value: value, Label: label, Sort: sort,
		Enabled: enabled, Remark: strings.TrimSpace(remark),
	}
	if err := s.repo.CreateItem(it); err != nil {
		return nil, err
	}
	return it, nil
}

func (s *Service) UpdateItem(id int64, label string, sort int, enabled bool, remark string) error {
	it, err := s.repo.GetItemByID(id)
	if err != nil {
		return err
	}
	if l := strings.TrimSpace(label); l != "" {
		it.Label = l
	}
	it.Sort = sort
	it.Enabled = enabled
	it.Remark = strings.TrimSpace(remark)
	return s.repo.UpdateItem(it)
}

func (s *Service) DeleteItem(id int64) error {
	it, err := s.repo.GetItemByID(id)
	if err != nil {
		return err
	}
	if it.Builtin {
		return badReq("内置项不可删除，可停用")
	}
	return s.repo.DeleteItem(id)
}
