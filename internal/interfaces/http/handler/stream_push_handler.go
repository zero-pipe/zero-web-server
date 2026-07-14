package handler

import (
	"context"
	"strconv"
	"time"

	streampushapp "zero-web-server/internal/application/streampush"
	"zero-web-server/internal/infrastructure/persistence/model"
	"zero-web-server/internal/interfaces/http/dto"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type StreamPushHandler struct {
	svc            *streampushapp.Service
	playTimeoutSec int
}

func NewStreamPushHandler(svc *streampushapp.Service, playTimeoutMs int) *StreamPushHandler {
	sec := playTimeoutMs / 1000
	if sec <= 0 {
		sec = 180
	}
	return &StreamPushHandler{svc: svc, playTimeoutSec: sec}
}

type streamPushBody struct {
	model.StreamPush
	GBDeviceID string `json:"gbDeviceId"`
	GBName     string `json:"gbName"`
	GbID       string `json:"gbId"`
	Name       string `json:"name"`
}

func (h *StreamPushHandler) List(c *gin.Context) {
	page, count := parsePageCount(c)
	var pushing *bool
	if v := c.Query("pushing"); v != "" {
		b := v == "true"
		pushing = &b
	}
	list, total, err := h.svc.List(page, count, c.Query("query"), pushing, c.Query("mediaServerId"))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, dto.NewPageInfo(list, total, page, count))
}

func (h *StreamPushHandler) Add(c *gin.Context) {
	var body streamPushBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	gbID := body.GBDeviceID
	if gbID == "" {
		gbID = body.GbID
	}
	name := body.GBName
	if name == "" {
		name = body.Name
	}
	if err := h.svc.Add(&body.StreamPush, gbID, name); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, body.StreamPush)
}

func (h *StreamPushHandler) Update(c *gin.Context) {
	var body model.StreamPush
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.Update(&body); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *StreamPushHandler) Remove(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("id"))
	if err := h.svc.Remove(id); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *StreamPushHandler) BatchRemove(c *gin.Context) {
	var body struct {
		IDs []int `json:"ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.BatchRemove(body.IDs); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *StreamPushHandler) SaveToGB(c *gin.Context) {
	var body struct {
		ID         int    `json:"id"`
		GBDeviceID string `json:"gbDeviceId"`
		Name       string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.SaveToGB(body.ID, body.GBDeviceID, body.Name); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *StreamPushHandler) Start(c *gin.Context) {
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

func (h *StreamPushHandler) RemoveFromGB(c *gin.Context) {
	var body struct {
		ID int `json:"id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.RemoveFromGB(body.ID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}
