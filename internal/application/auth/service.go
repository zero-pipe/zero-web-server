package auth

import (
	"errors"
	"fmt"

	domainuser "zero-web-kit/internal/domain/user"
	jwtmgr "zero-web-kit/pkg/jwt"
)

var ErrInvalidCredentials = errors.New("用户名或密码错误")

type Service struct {
	userRepo  domainuser.Repository
	jwt       *jwtmgr.Manager
	serverID  string
}

func NewService(userRepo domainuser.Repository, jwt *jwtmgr.Manager, serverID string) *Service {
	return &Service{
		userRepo: userRepo,
		jwt:      jwt,
		serverID: serverID,
	}
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

	return domainuser.NewLoginUser(u, token, s.serverID), token, nil
}

func (s *Service) GetUserInfo(username string) (*domainuser.LoginUser, error) {
	u, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return nil, err
	}
	return domainuser.NewLoginUser(u, "", s.serverID), nil
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
