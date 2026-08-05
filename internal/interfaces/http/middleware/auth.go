package middleware

import (
	"net/http"
	"strings"

	"zero-web-server/internal/application/rbac"
	domainuser "zero-web-server/internal/domain/user"
	jwtmgr "zero-web-server/pkg/jwt"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

const (
	ContextUserKey  = "username"
	ContextRoleID   = "roleId"
	ContextMenusKey = "menus"
)

func Auth(jwtManager *jwtmgr.Manager, userRepo domainuser.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader(jwtmgr.Header)
		if token == "" {
			token = c.Query("access-token")
		}
		if token == "" {
			c.JSON(http.StatusUnauthorized, response.Fail(response.CodeUnauth, "未登录"))
			c.Abort()
			return
		}

		claims, err := jwtManager.ParseToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, response.Fail(response.CodeUnauth, "登录已过期"))
			c.Abort()
			return
		}

		username := strings.TrimSpace(claims.UserName)
		c.Set(ContextUserKey, username)

		if userRepo != nil && username != "" {
			if u, err := userRepo.FindByUsername(username); err == nil && u != nil && u.Role != nil {
				menus := rbac.ParseMenus(u.Role.ID, u.Role.Authority)
				c.Set(ContextRoleID, u.Role.ID)
				c.Set(ContextMenusKey, menus)
			} else {
				c.Set(ContextRoleID, 0)
				c.Set(ContextMenusKey, []string{})
			}
		}

		c.Next()
	}
}

// RequireMenu 要求当前用户拥有指定菜单权限之一。
func RequireMenu(codes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		menus := GetMenus(c)
		if rbac.HasAny(menus, codes...) {
			c.Next()
			return
		}
		response.Error(c, response.CodeForbidden, "无权限访问")
		c.Abort()
	}
}

// RequireAdmin 仅管理员角色（role_id=1）。
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if GetRoleID(c) == 1 {
			c.Next()
			return
		}
		response.Error(c, response.CodeForbidden, "需要管理员权限")
		c.Abort()
	}
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, access-token, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func OptionalAuth(jwtManager *jwtmgr.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader(jwtmgr.Header)
		if token == "" {
			token = c.Query("access-token")
		}
		if token != "" {
			if claims, err := jwtManager.ParseToken(token); err == nil {
				c.Set(ContextUserKey, claims.UserName)
			}
		}
		c.Next()
	}
}

func GetUsername(c *gin.Context) string {
	v, _ := c.Get(ContextUserKey)
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func GetRoleID(c *gin.Context) int {
	v, _ := c.Get(ContextRoleID)
	id, _ := v.(int)
	return id
}

func GetMenus(c *gin.Context) []string {
	v, ok := c.Get(ContextMenusKey)
	if !ok || v == nil {
		return []string{}
	}
	menus, _ := v.([]string)
	if menus == nil {
		return []string{}
	}
	return menus
}
