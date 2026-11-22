package handler

import (
	"strconv"
	"strings"

	accesstokenapp "zero-web-server/internal/application/accesstoken"
	"zero-web-server/internal/dto"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

func parseInt64Query(c *gin.Context, key string, def int64) int64 {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return def
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return def
	}
	return v
}

// AccessTokenHandler 对外接入密钥可视化配置。
type AccessTokenHandler struct {
	svc *accesstokenapp.Service
}

func NewAccessTokenHandler(svc *accesstokenapp.Service) *AccessTokenHandler {
	return &AccessTokenHandler{svc: svc}
}

func (h *AccessTokenHandler) List(c *gin.Context) {
	page, count := parsePageCount(c)
	list, total, err := h.svc.List(page, count)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, dto.NewPageInfo(list, total, page, count))
}

// First 返回首个启用中的 token（供推/拉流 URL 拼装）。
func (h *AccessTokenHandler) First(c *gin.Context) {
	response.OK(c, gin.H{"token": h.svc.FirstEnabled()})
}

func (h *AccessTokenHandler) Add(c *gin.Context) {
	name := c.Query("name")
	token := c.Query("token")
	remark := c.Query("remark")
	enabled := c.Query("enabled") != "0"
	row, err := h.svc.Create(name, token, remark, enabled)
	if err != nil {
		if accesstokenapp.IsBadRequest(err) {
			response.Error(c, response.CodeBadReq, err.Error())
			return
		}
		response.FromError(c, err)
		return
	}
	response.OK(c, row)
}

func (h *AccessTokenHandler) Update(c *gin.Context) {
	id := parseInt64Query(c, "id", 0)
	if id <= 0 {
		response.Error(c, response.CodeBadReq, "id 无效")
		return
	}
	name := c.Query("name")
	remark := c.Query("remark")
	enabled := c.Query("enabled") != "0"
	if err := h.svc.Update(id, name, remark, enabled); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *AccessTokenHandler) Enable(c *gin.Context) {
	h.setEnabled(c, true)
}

func (h *AccessTokenHandler) Disable(c *gin.Context) {
	h.setEnabled(c, false)
}

func (h *AccessTokenHandler) setEnabled(c *gin.Context, enabled bool) {
	id := parseInt64Query(c, "id", 0)
	if id <= 0 {
		response.Error(c, response.CodeBadReq, "id 无效")
		return
	}
	if err := h.svc.SetEnabled(id, enabled); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *AccessTokenHandler) Reset(c *gin.Context) {
	id := parseInt64Query(c, "id", 0)
	if id <= 0 {
		response.Error(c, response.CodeBadReq, "id 无效")
		return
	}
	token, err := h.svc.Reset(id)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"token": token})
}

func (h *AccessTokenHandler) Delete(c *gin.Context) {
	id := parseInt64Query(c, "id", 0)
	if id <= 0 {
		response.Error(c, response.CodeBadReq, "id 无效")
		return
	}
	if err := h.svc.Delete(id); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}
