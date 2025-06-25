package user

type Role struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Authority  string `json:"authority"`
	CreateTime string `json:"createTime"`
	UpdateTime string `json:"updateTime"`
}

type User struct {
	ID         int    `json:"id"`
	Username   string `json:"username"`
	Password   string `json:"password,omitempty"`
	CreateTime string `json:"createTime"`
	UpdateTime string `json:"updateTime"`
	PushKey    string `json:"pushKey"`
	Role       *Role  `json:"role"`
}

type LoginUser struct {
	Username    string `json:"username"`
	AccessToken string `json:"accessToken"`
	ServerID    string `json:"serverId"`
	PushKey     string `json:"pushKey,omitempty"`
	Role        *Role  `json:"role,omitempty"`
}

func NewLoginUser(u *User, token, serverID string) *LoginUser {
	lu := &LoginUser{
		Username:    u.Username,
		AccessToken: token,
		ServerID:    serverID,
		PushKey:     u.PushKey,
	}
	if u.Role != nil {
		lu.Role = u.Role
	}
	return lu
}

type Repository interface {
	FindByUsername(username string) (*User, error)
	FindByID(id int) (*User, error)
	List(page, count int) ([]*User, int64, error)
	Create(username, password string, roleID int) error
	Delete(id int) error
	UpdatePassword(id int, password string) error
	ChangePushKey(id int, pushKey string) error
}
