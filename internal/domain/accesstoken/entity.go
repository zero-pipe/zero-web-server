package accesstoken

import "errors"

// ErrNotFound 表示接入 token 不存在。
var ErrNotFound = errors.New("access token not found")

// AccessToken 对外接入密钥（领域实体）。
//
// 用于外部集成方携带 ?token= 推/拉流或调用开放接口。平台只做“存在 + 启用”校验，
// 不区分接口权限、不做限流。一条记录对应一个集成方，便于单独启用/停用/吊销。
type AccessToken struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`   // 集成方名称，如“南门摄像头供应商”
	Token      string `json:"token"`  // 实际携带的密钥串
	Enabled    bool   `json:"enabled"` // 是否启用（停用即失效）
	Remark     string `json:"remark"`
	CreateTime string `json:"createTime"`
	UpdateTime string `json:"updateTime"`
}

type Repository interface {
	List(page, count int) ([]AccessToken, int64, error)
	GetByID(id int64) (*AccessToken, error)
	Create(t *AccessToken) error
	Update(t *AccessToken) error
	Delete(id int64) error
	FindEnabledByToken(token string) (*AccessToken, error)
	ListEnabledTokens() ([]string, error)
}
