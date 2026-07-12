package handler

import (
	"net/http"
	"strconv"

	appauth "zero-web-kit/internal/application/auth"
	"zero-web-kit/internal/application/ops"
	"zero-web-kit/pkg/jwt"
	"zero-web-kit/pkg/response"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	auth *appauth.Service
}

func NewUserHandler(auth *appauth.Service) *UserHandler {
	return &UserHandler{auth: auth}
}

func (h *UserHandler) Login(c *gin.Context) {
	username := c.Query("username")
	password := c.PostForm("password")
	if password == "" {
		password = c.Query("password")
	}
	if username == "" || password == "" {
		response.Error(c, response.CodeBadReq, "用户名和密码不能为空")
		return
	}

	loginUser, token, err := h.auth.Login(username, password)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}

	c.Header(jwt.Header, token)
	response.OK(c, loginUser)
}

func (h *UserHandler) Logout(c *gin.Context) {
	c.Status(http.StatusOK)
}

func (h *UserHandler) UserInfo(c *gin.Context) {
	username, _ := c.Get("username")
	name, _ := username.(string)
	loginUser, err := h.auth.GetUserInfo(name)
	if err != nil {
		response.Error(c, response.CodeError, "获取用户信息失败")
		return
	}
	response.OK(c, loginUser)
}

type HealthHandler struct {
	version string
}

func NewHealthHandler(version string) *HealthHandler {
	return &HealthHandler{version: version}
}

func (h *HealthHandler) Version(c *gin.Context) {
	response.OK(c, gin.H{"version": h.version, "name": "zero-web-kit"})
}

func (h *HealthHandler) Health(c *gin.Context) {
	response.OK(c, gin.H{"status": "ok"})
}

type ServerHandler struct {
	serverID string
	metrics  *ops.Metrics
}

func NewServerHandler(serverID string, metrics *ops.Metrics) *ServerHandler {
	if metrics == nil {
		metrics = ops.DefaultMetrics
	}
	return &ServerHandler{serverID: serverID, metrics: metrics}
}

func (h *ServerHandler) Config(c *gin.Context) {
	response.OK(c, gin.H{
		"serverId": h.serverID,
	})
}

func (h *ServerHandler) SystemInfo(c *gin.Context) {
	response.OK(c, h.metrics.Snapshot())
}

func parsePageCount(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "15"))
	return page, count
}
