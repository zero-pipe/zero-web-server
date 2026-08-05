package auth

import (
	"errors"
	"fmt"
	"strings"

	"zero-web-server/internal/application/rbac"
	domainuser "zero-web-server/internal/domain/user"
	jwtmgr "zero-web-server/pkg/jwt"
)

var ErrInvalidCredentials = errors.New("用户名或密码错误")

type Service struct {
	userRepo domainuser.Repository
	jwt      *jwtmgr.Manager
	serverID string
}

func NewService(userRepo domainuser.Repository, jwt *jwtmgr.Manager, serverID string) *Service {
	return &Service{
		userRepo: userRepo,
		jwt:      jwt,
		serverID: serverID,
	}
}

func (s *Service) loginUserFrom(u *domainuser.User, token string) *domainuser.LoginUser {
	menus := []string{}
	if u.Role != nil {
		menus = rbac.ParseMenus(u.Role.ID, u.Role.Authority)
	}
	return domainuser.NewLoginUser(u, token, s.serverID, menus)
}

func (s *Service) Login(username, passwordMD5 string) (*domainuser.LoginUser, string, error) {
	u, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}
	if u.Password != passwordMD5 {
		return nil, "", ErrInvalidCredentials
	}

	token, err := s.jwt.CreateToken(username)
	if err != nil {
		return nil, "", err
	}

	return s.loginUserFrom(u, token), token, nil
}

func (s *Service) GetUserInfo(username string) (*domainuser.LoginUser, error) {
	u, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return nil, err
	}
	return s.loginUserFrom(u, ""), nil
}

func (s *Service) ResolveMenus(username string) (roleID int, menus []string, err error) {
	u, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return 0, nil, err
	}
	if u.Role == nil {
		return 0, []string{}, nil
	}
	return u.Role.ID, rbac.ParseMenus(u.Role.ID, u.Role.Authority), nil
}

func (s *Service) ListUsers(page, count int) ([]*domainuser.User, int64, error) {
	return s.userRepo.List(page, count)
}

func (s *Service) AddUser(username, password string, roleID int) error {
	if _, err := s.userRepo.FindByUsername(username); err == nil {
		return fmt.Errorf("用户名已存在")
	}
	return s.userRepo.Create(username, password, roleID)
}

func (s *Service) DeleteUser(id int) error {
	if id == 1 {
		return fmt.Errorf("不能删除默认管理员")
	}
	return s.userRepo.Delete(id)
}

func (s *Service) ChangePassword(username, oldPassword, newPassword string) error {
	u, err := s.userRepo.FindByUsername(username)
	if err != nil || u.Password != oldPassword {
		return ErrInvalidCredentials
	}
	return s.userRepo.UpdatePassword(u.ID, newPassword)
}

func (s *Service) ChangePasswordForAdmin(userID int, password string) error {
	return s.userRepo.UpdatePassword(userID, password)
}

func (s *Service) ChangePushKey(userID int, pushKey string) error {
	return s.userRepo.ChangePushKey(userID, pushKey)
}

func (s *Service) ListRoles() ([]*domainuser.Role, error) {
	return s.userRepo.ListRoles()
}

func (s *Service) AddRole(name string, menus []string) (*domainuser.Role, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("角色名称不能为空")
	}
	return s.userRepo.CreateRole(name, rbac.EncodeMenus(menus))
}

func (s *Service) UpdateRole(id int, name string, menus []string) error {
	if id <= 0 {
		return fmt.Errorf("角色无效")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("角色名称不能为空")
	}
	authority := rbac.EncodeMenus(menus)
	if id == 1 {
		authority = rbac.AuthorityAll // 管理员始终全权限
	}
	return s.userRepo.UpdateRole(id, name, authority)
}

func (s *Service) DeleteRole(id int) error {
	if id == 1 {
		return fmt.Errorf("不能删除管理员角色")
	}
	n, err := s.userRepo.CountUsersByRole(id)
	if err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("角色下仍有用户，无法删除")
	}
	return s.userRepo.DeleteRole(id)
}
