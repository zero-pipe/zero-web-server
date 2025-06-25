package handler

import (
	"strconv"

	platformapp "zero-web-kit/internal/application/platform"
	domainplatform "zero-web-kit/internal/domain/platform"
	"zero-web-kit/internal/interfaces/http/dto"
	"zero-web-kit/pkg/response"

	"github.com/gin-gonic/gin"
)

type PlatformHandler struct {
	svc     *platformapp.Service
	channel *platformapp.ChannelService
}

func NewPlatformHandler(svc *platformapp.Service, channel *platformapp.ChannelService) *PlatformHandler {
	return &PlatformHandler{svc: svc, channel: channel}
}

func (h *PlatformHandler) ServerConfig(c *gin.Context) {
	response.OK(c, h.svc.ServerConfig())
}

func (h *PlatformHandler) Query(c *gin.Context) {
	page, count := parsePageCount(c)
	list, total, err := h.svc.List(page, count, c.Query("query"))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, dto.NewPageInfo(list, total, page, count))
}

func (h *PlatformHandler) Add(c *gin.Context) {
	var p domainplatform.Platform
	if err := c.ShouldBindJSON(&p); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.Add(&p); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PlatformHandler) Update(c *gin.Context) {
	var p domainplatform.Platform
	if err := c.ShouldBindJSON(&p); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.Update(&p); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PlatformHandler) Delete(c *gin.Context) {
	id := parseIntQuery(c, "id", 0)
	if id <= 0 {
		response.Error(c, response.CodeBadReq, "id无效")
		return
	}
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PlatformHandler) Exit(c *gin.Context) {
	if err := h.svc.Exit(c.Param("deviceGbId")); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PlatformHandler) PushChannel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("id"))
	if err := h.channel.PushCatalog(id); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PlatformHandler) ChannelList(c *gin.Context) {
	page, count := parsePageCount(c)
	platformID, _ := strconv.Atoi(c.Query("platformId"))
	var hasShare *bool
	if v := c.Query("hasShare"); v != "" {
		b := v == "true"
		hasShare = &b
	}
	list, total, err := h.channel.List(platformID, page, count, c.Query("query"), hasShare)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, dto.NewPageInfo(list, total, page, count))
}

type channelBatchReq struct {
	PlatformID int   `json:"platformId"`
	ChannelIDs []int `json:"channelIds"`
	All        bool  `json:"all"`
}

type deviceBatchReq struct {
	PlatformID int      `json:"platformId"`
	DeviceIDs  []string `json:"deviceIds"`
}

func (h *PlatformHandler) ChannelAdd(c *gin.Context) {
	var req channelBatchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.channel.AddChannels(req.PlatformID, req.ChannelIDs, req.All); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PlatformHandler) ChannelDeviceAdd(c *gin.Context) {
	var req deviceBatchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.channel.AddChannelsByDevice(req.PlatformID, req.DeviceIDs); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PlatformHandler) ChannelDeviceRemove(c *gin.Context) {
	var req deviceBatchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.channel.RemoveChannelsByDevice(req.PlatformID, req.DeviceIDs); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PlatformHandler) ChannelRemove(c *gin.Context) {
	var req channelBatchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.channel.RemoveChannels(req.PlatformID, req.ChannelIDs, req.All); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PlatformHandler) ChannelCustomUpdate(c *gin.Context) {
	response.OK(c, nil)
}
