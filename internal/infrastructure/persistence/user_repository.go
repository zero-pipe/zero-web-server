package persistence

import (
	"crypto/md5"
	"encoding/hex"
	"time"

	domainuser "zero-web-server/internal/domain/user"
	"zero-web-server/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByUsername(username string) (*domainuser.User, error) {
	var m model.User
	if err := r.db.Preload("Role").Where("username = ?", username).First(&m).Error; err != nil {
		return nil, err
	}
	return toDomainUser(&m), nil
}

func (r *UserRepository) FindByID(id int) (*domainuser.User, error) {
	var m model.User
	if err := r.db.Preload("Role").Where("id = ?", id).First(&m).Error; err != nil {
		return nil, err
	}
	return toDomainUser(&m), nil
}

func (r *UserRepository) ListAll() ([]*domainuser.User, error) {
	var rows []model.User
	if err := r.db.Preload("Role").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainuser.User, len(rows))
	for i := range rows {
		out[i] = toDomainUser(&rows[i])
	}
	return out, nil
}

func (r *UserRepository) CheckPushAuthority(callID, sign string) bool {
	users, err := r.ListAll()
	if err != nil || len(users) == 0 {
		return false
	}
	for _, u := range users {
		if u.PushKey == "" {
			continue
		}
		checkStr := u.PushKey
		if callID != "" {
			checkStr = callID + "_" + u.PushKey
		}
		if md5Hex(checkStr) == sign {
			return true
		}
	}
	return false
}

func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func (r *UserRepository) List(page, count int) ([]*domainuser.User, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Model(&model.User{})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.User
	if err := q.Preload("Role").Order("id ASC").Offset((page - 1) * count).Limit(count).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*domainuser.User, len(rows))
	for i := range rows {
		out[i] = toDomainUser(&rows[i])
	}
	return out, total, nil
}

func (r *UserRepository) Create(username, password string, roleID int) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	return r.db.Create(&model.User{
		Username: username, Password: password, RoleID: roleID,
		CreateTime: now, UpdateTime: now,
	}).Error
}

func (r *UserRepository) Delete(id int) error {
	return r.db.Delete(&model.User{}, id).Error
}

func (r *UserRepository) UpdatePassword(id int, password string) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Updates(map[string]any{
		"password": password, "update_time": time.Now().Format("2006-01-02 15:04:05"),
	}).Error
}

func (r *UserRepository) ChangePushKey(id int, pushKey string) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Updates(map[string]any{
		"push_key": pushKey, "update_time": time.Now().Format("2006-01-02 15:04:05"),
	}).Error
}

func (r *UserRepository) ListRoles() ([]*domainuser.Role, error) {
	var rows []model.UserRole
	if err := r.db.Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainuser.Role, len(rows))
	for i := range rows {
		out[i] = toDomainRole(&rows[i])
	}
	return out, nil
}

func (r *UserRepository) CreateRole(name, authority string) (*domainuser.Role, error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	m := &model.UserRole{
		Name: name, Authority: authority,
		CreateTime: now, UpdateTime: now,
	}
	if err := r.db.Create(m).Error; err != nil {
		return nil, err
	}
	return toDomainRole(m), nil
}

func (r *UserRepository) UpdateRole(id int, name, authority string) error {
	return r.db.Model(&model.UserRole{}).Where("id = ?", id).Updates(map[string]any{
		"name": name, "authority": authority,
		"update_time": time.Now().Format("2006-01-02 15:04:05"),
	}).Error
}

func (r *UserRepository) DeleteRole(id int) error {
	return r.db.Delete(&model.UserRole{}, id).Error
}

func (r *UserRepository) CountUsersByRole(roleID int) (int64, error) {
	var n int64
	err := r.db.Model(&model.User{}).Where("role_id = ?", roleID).Count(&n).Error
	return n, err
}

func toDomainRole(m *model.UserRole) *domainuser.Role {
	return &domainuser.Role{
		ID:         m.ID,
		Name:       m.Name,
		Authority:  m.Authority,
		CreateTime: m.CreateTime,
		UpdateTime: m.UpdateTime,
	}
}

func toDomainUser(m *model.User) *domainuser.User {
	u := &domainuser.User{
		ID:         m.ID,
		Username:   m.Username,
		Password:   m.Password,
		CreateTime: m.CreateTime,
		UpdateTime: m.UpdateTime,
		PushKey:    m.PushKey,
	}
	if m.Role.ID > 0 {
		u.Role = toDomainRole(&m.Role)
	}
	return u
}
