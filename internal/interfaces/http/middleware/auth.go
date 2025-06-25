package middleware

import (
	"net/http"
	"strings"

	jwtmgr "zero-web-kit/pkg/jwt"
	"zero-web-kit/pkg/response"

	"github.com/gin-gonic/gin"
)

const ContextUserKey = "username"

func Auth(jwtManager *jwtmgr.Manager) gin.HandlerFunc {
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

		c.Set(ContextUserKey, claims.UserName)
		c.Next()
	}
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, access-token, Authorization")
		if c.Request.Method == http.MethodOptions {
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
