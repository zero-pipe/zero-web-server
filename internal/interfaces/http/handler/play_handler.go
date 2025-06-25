package handler

import (
	"context"
	"time"

	playapp "zero-web-kit/internal/application/play"
	"zero-web-kit/pkg/response"

	"github.com/gin-gonic/gin"
)

type PlayHandler struct {
	svc            *playapp.Service
	playTimeoutSec int
}

func NewPlayHandler(svc *playapp.Service, playTimeoutMs int) *PlayHandler {
	sec := playTimeoutMs / 1000
	if sec <= 0 {
		sec = 180
	}
	return &PlayHandler{svc: svc, playTimeoutSec: sec}
}

func (h *PlayHandler) Start(c *gin.Context) {
	deviceID := c.Param("deviceId")
	channelID := c.Param("channelId")
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(h.playTimeoutSec)*time.Second)
	defer cancel()

	content, err := h.svc.StartPlay(ctx, deviceID, channelID)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, content)
}

func (h *PlayHandler) Stop(c *gin.Context) {
	if err := h.svc.StopPlay(c.Param("deviceId"), c.Param("channelId")); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PlayHandler) Broadcast(c *gin.Context) {
	broadcastMode := c.Query("broadcastMode") == "true"
	result, err := h.svc.AudioBroadcast(c.Param("deviceId"), c.Param("channelId"), broadcastMode)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *PlayHandler) BroadcastStop(c *gin.Context) {
	if err := h.svc.StopAudioBroadcast(c.Param("deviceId"), c.Param("channelId")); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}
