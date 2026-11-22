package accesstokenapp

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	domainaccesstoken "zero-web-server/internal/domain/accesstoken"
)

// Service 接入 token 业务逻辑：可视化配置 + “存在且启用”校验。
//
// 只做 token 是否有效的判断，不做接口级权限、不限流。
type Service struct {
	repo domainaccesstoken.Repository

	// onChange 在 token 集合变化后回调（用于把启用中的 token 同步到媒体节点）。
	onChange func(tokens []string)
}

func NewService(repo domainaccesstoken.Repository) *Service {
	return &Service{repo: repo}
}

// SetOnChange 注册变更回调（媒体节点 token 同步）。
func (s *Service) SetOnChange(fn func(tokens []string)) {
	s.onChange = fn
}

// List 分页查询。
func (s *Service) List(page, count int) ([]domainaccesstoken.AccessToken, int64, error) {
	return s.repo.List(page, count)
}

// Create 新建接入 token；token 串留空时自动生成。
func (s *Service) Create(name, token, remark string, enabled bool) (*domainaccesstoken.AccessToken, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "未命名"
	}
	token = strings.TrimSpace(token)
	if token == "" {
		gen, err := generateToken()
		if err != nil {
			return nil, err
		}
		token = gen
	}
	if len(token) > 128 {
		return nil, errBadReq("token 长度不能超过 128")
	}
	row := &domainaccesstoken.AccessToken{
		Name:    name,
		Token:   token,
		Enabled: enabled,
		Remark:  strings.TrimSpace(remark),
	}
	if err := s.repo.Create(row); err != nil {
		return nil, err
	}
	s.notify()
	return row, nil
}

// Update 修改名称 / 备注 / 启用状态（不改 token 本身；重置走 Reset）。
func (s *Service) Update(id int64, name, remark string, enabled bool) error {
	row, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if n := strings.TrimSpace(name); n != "" {
		row.Name = n
	}
	row.Remark = strings.TrimSpace(remark)
	row.Enabled = enabled
	if err := s.repo.Update(row); err != nil {
		return err
	}
	s.notify()
	return nil
}

// SetEnabled 启用 / 停用。
func (s *Service) SetEnabled(id int64, enabled bool) error {
	row, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	row.Enabled = enabled
	if err := s.repo.Update(row); err != nil {
		return err
	}
	s.notify()
	return nil
}

// Reset 重新生成 token 串。
func (s *Service) Reset(id int64) (string, error) {
	row, err := s.repo.GetByID(id)
	if err != nil {
		return "", err
	}
	gen, err := generateToken()
	if err != nil {
		return "", err
	}
	row.Token = gen
	if err := s.repo.Update(row); err != nil {
		return "", err
	}
	s.notify()
	return gen, nil
}

// Delete 删除。
func (s *Service) Delete(id int64) error {
	if err := s.repo.Delete(id); err != nil {
		return err
	}
	s.notify()
	return nil
}

// Verify 校验 token 是否有效（存在且启用）。空串一律无效。
func (s *Service) Verify(token string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return false
	}
	_, err := s.repo.FindEnabledByToken(token)
	return err == nil
}

// EnabledTokens 返回全部启用中的 token 串。
func (s *Service) EnabledTokens() []string {
	tokens, err := s.repo.ListEnabledTokens()
	if err != nil {
		return nil
	}
	return tokens
}

// FirstEnabled 返回首个启用中的 token（供内部页面拼推/拉流 URL 使用）；无则空串。
func (s *Service) FirstEnabled() string {
	tokens := s.EnabledTokens()
	if len(tokens) == 0 {
		return ""
	}
	return tokens[0]
}

func (s *Service) notify() {
	if s.onChange == nil {
		return
	}
	// 即使为空也要推（空表示清除 ZMS 白名单并关闭验签）。
	s.onChange(s.EnabledTokens())
}

type badReq string

func (e badReq) Error() string { return string(e) }

func errBadReq(msg string) error { return badReq(msg) }

// IsBadRequest 供 handler 判断是否为参数错误。
func IsBadRequest(err error) bool {
	_, ok := err.(badReq)
	return ok
}

func generateToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
