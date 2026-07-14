package handler

import (
	deviceaccess "zero-web-server/internal/application/deviceaccess"
	"zero-web-server/internal/interfaces/http/dto"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type DeviceAccessHandler struct {
	svc *deviceaccess.Service
}

func NewDeviceAccessHandler(svc *deviceaccess.Service) *DeviceAccessHandler {
	return &DeviceAccessHandler{svc: svc}
}

func (h *DeviceAccessHandler) List(c *gin.Context) {
	page, count := parsePageCount(c)
	list, total, err := h.svc.List(c.Request.Context(), deviceaccess.ListQuery{
		Page:       page,
		Count:      count,
		Query:      c.Query("query"),
		AccessMode: c.Query("accessMode"),
		Protocol:   c.Query("protocol"),
		Status:     c.Query("status"),
	})
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, dto.NewPageInfo(list, total, page, count))
}

func (h *DeviceAccessHandler) Create(c *gin.Context) {
	var req deviceaccess.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	view, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, view)
}

// Update PUT /api/devices?id=gb:xxx
func (h *DeviceAccessHandler) Update(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		response.Error(c, response.CodeBadReq, "设备ID为空")
		return
	}
	var req deviceaccess.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	view, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, view)
}

// Delete POST /api/devices/delete  body: {"id":"gb:xxx"}（兼 query）
func (h *DeviceAccessHandler) Delete(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		var body struct {
			ID string `json:"id"`
		}
		_ = c.ShouldBindJSON(&body)
		id = body.ID
	}
	if id == "" {
		response.Error(c, response.CodeBadReq, "设备ID为空")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

// Sync POST /api/devices/sync?id=gb:xxx
func (h *DeviceAccessHandler) Sync(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		response.Error(c, response.CodeBadReq, "设备ID为空")
		return
	}
	if err := h.svc.Sync(c.Request.Context(), id); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}
