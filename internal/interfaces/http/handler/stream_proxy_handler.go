package handler

import (
	"context"
	"strconv"
	"time"

	streamproxyapp "zero-web-server/internal/application/streamproxy"
	"zero-web-server/internal/infrastructure/persistence/model"
	"zero-web-server/internal/interfaces/http/dto"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type StreamProxyHandler struct {
	svc            *streamproxyapp.Service
	playTimeoutSec int
}

func NewStreamProxyHandler(svc *streamproxyapp.Service, playTimeoutMs int) *StreamProxyHandler {
	sec := playTimeoutMs / 1000
	if sec <= 0 {
		sec = 180
	}
	return &StreamProxyHandler{svc: svc, playTimeoutSec: sec}
}

type streamProxyBody struct {
	model.StreamProxy
	GBDeviceID string `json:"gbDeviceId"`
	GbID       string `json:"gbId"`
}

func (h *StreamProxyHandler) List(c *gin.Context) {
	page, count := parsePageCount(c)
	var pulling *bool
	if v := c.Query("pulling"); v != "" {
		b := v == "true"
		pulling = &b
	}
	list, total, err := h.svc.List(page, count, c.Query("query"), pulling, c.Query("mediaServerId"))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, dto.NewPageInfo(list, total, page, count))
}

func (h *StreamProxyHandler) Add(c *gin.Context) {
	var body streamProxyBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	gbID := body.GBDeviceID
	if gbID == "" {
		gbID = body.GbID
	}
	if err := h.svc.Add(&body.StreamProxy, gbID, body.Name); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, body.StreamProxy)
}

func (h *StreamProxyHandler) Update(c *gin.Context) {
	var body model.StreamProxy
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.Update(&body); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, body)
}

func (h *StreamProxyHandler) Save(c *gin.Context) {
	h.Add(c)
}

func (h *StreamProxyHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("id"))
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *StreamProxyHandler) Start(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("id"))
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(h.playTimeoutSec)*time.Second)
	defer cancel()
	content, err := h.svc.Start(ctx, id)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, content)
}

func (h *StreamProxyHandler) Stop(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("id"))
	if err := h.svc.Stop(id); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *StreamProxyHandler) FFmpegCmdList(c *gin.Context) {
	response.OK(c, h.svc.FFmpegCmdList(c.Query("mediaServerId")))
}
