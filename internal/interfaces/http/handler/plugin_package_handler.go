package handler

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	pluginpkgapp "zero-web-server/internal/application/pluginpkgapp"
	domainpkg "zero-web-server/internal/domain/pluginpkg"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

// PluginPackageHandler 管理插件安装包上传、下载与安装/卸载。
type PluginPackageHandler struct {
	svc   *pluginpkgapp.Service
	token string
}

func NewPluginPackageHandler(svc *pluginpkgapp.Service, agentToken string) *PluginPackageHandler {
	return &PluginPackageHandler{svc: svc, token: strings.TrimSpace(agentToken)}
}

// checkAgentToken 校验 Agent 下载凭据（与注册同一 token）。
func (h *PluginPackageHandler) checkAgentToken(c *gin.Context) bool {
	if h.token == "" {
		return true
	}
	if c.GetHeader("X-Agent-Token") != h.token {
		response.Error(c, response.CodeUnauth, "Agent 令牌无效")
		return false
	}
	return true
}

func (h *PluginPackageHandler) List(c *gin.Context) {
	list, err := h.svc.List()
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"list": list, "total": len(list)})
}

func (h *PluginPackageHandler) Get(c *gin.Context) {
	p, err := h.svc.Get(c.Param("id"))
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, p)
}

// Upload POST /api/v1/plugin-packages（multipart，字段 file）。
func (h *PluginPackageHandler) Upload(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		response.Error(c, response.CodeBadReq, "请上传文件字段 file")
		return
	}
	if fh.Size <= 0 || fh.Size > domainpkg.MaxPackageBytes {
		response.Error(c, response.CodeBadReq, fmt.Sprintf("文件大小需在 1B ~ %dMB 之间", domainpkg.MaxPackageBytes>>20))
		return
	}
	f, err := fh.Open()
	if err != nil {
		response.Error(c, response.CodeBadReq, "打开上传文件失败")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, domainpkg.MaxPackageBytes+1))
	if err != nil {
		response.Error(c, response.CodeBadReq, "读取上传文件失败")
		return
	}
	p, err := h.svc.Upload(c.Request.Context(), data)
	if err != nil {
		response.Error(c, response.CodeBadReq, err.Error())
		return
	}
	response.OK(c, p)
}

// Download GET /api/v1/plugin-packages/:id/download（Agent 专用，X-Agent-Token）。
func (h *PluginPackageHandler) Download(c *gin.Context) {
	if !h.checkAgentToken(c) {
		return
	}
	p, rc, err := h.svc.Open(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondErr(c, err)
		return
	}
	defer rc.Close()
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", strconv.FormatInt(p.SizeBytes, 10))
	c.Header("X-Package-SHA256", p.SHA256)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%s.zwplugin", p.PluginID, p.Version))
	if _, err := io.Copy(c.Writer, rc); err != nil {
		// 头已发送，只能中断连接；Agent 会因长度不符校验失败。
		return
	}
}

func (h *PluginPackageHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, nil)
}

// Install POST /api/v1/plugin-packages/:id/install?agentId=
func (h *PluginPackageHandler) Install(c *gin.Context) {
	agentID := strings.TrimSpace(c.Query("agentId"))
	if agentID == "" {
		var body struct {
			AgentID string `json:"agentId"`
		}
		if err := c.ShouldBindJSON(&body); err == nil {
			agentID = strings.TrimSpace(body.AgentID)
		}
	}
	in, err := h.svc.Install(c.Request.Context(), c.Param("id"), agentID, downloadBase(c))
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, in)
}

// ListInstalls GET /api/v1/plugin-installs?agentId=
func (h *PluginPackageHandler) ListInstalls(c *gin.Context) {
	list, err := h.svc.ListInstalls(c.Query("agentId"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"list": list, "total": len(list)})
}

// Uninstall DELETE /api/v1/plugin-installs/:id
func (h *PluginPackageHandler) Uninstall(c *gin.Context) {
	if err := h.svc.Uninstall(c.Request.Context(), c.Param("id")); err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *PluginPackageHandler) respondErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainpkg.ErrNotFound):
		response.Error(c, response.CodeBadReq, "记录不存在")
	case errors.Is(err, pluginpkgapp.ErrPackageInUse):
		response.Error(c, response.CodeBadReq, err.Error())
	default:
		response.FromError(c, err)
	}
}

// downloadBase 生成 Agent 拉包使用的平台基址（优先显式配置，其次请求 Host）。
func downloadBase(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	if host := c.GetHeader("X-Forwarded-Host"); host != "" {
		return scheme + "://" + host
	}
	return scheme + "://" + c.Request.Host
}
