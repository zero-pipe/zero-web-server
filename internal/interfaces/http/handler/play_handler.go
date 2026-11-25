package handler

import (
	"context"
	"time"

	assetapp "zero-web-server/internal/application/asset"
	playapp "zero-web-server/internal/application/play"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type PlayHandler struct {
	svc            *playapp.Service
	assets         *assetapp.Service
	playTimeoutSec int
}

func NewPlayHandler(svc *playapp.Service, assets *assetapp.Service, playTimeoutMs int) *PlayHandler {
	sec := playTimeoutMs / 1000
	if sec <= 0 {
		sec = 180
	}
	return &PlayHandler{svc: svc, assets: assets, playTimeoutSec: sec}
}

func (h *PlayHandler) Start(c *gin.Context) {
	deviceID := c.Param("deviceId")
	channelID := c.Param("channelId")
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(h.playTimeoutSec)*time.Second)
	defer cancel()

	content, err := h.svc.StartPlay(ctx, deviceID, channelID)
	if err != nil && h.assets != nil {
		// 国标取流失败时，若客户端给的是通道内码（ONVIF/插件等非国标设备没有国标码），
		// 回退到统一资源取流，保证客户端对三类设备都能播。
		if content2, err2 := h.assets.StartLiveByChannelInternalCode(ctx, channelID); err2 == nil {
			response.OK(c, content2)
			return
		}
	}
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, content)
}

func (h *PlayHandler) Stop(c *gin.Context) {
	deviceID := c.Param("deviceId")
	channelID := c.Param("channelId")
	if err := h.svc.StopPlay(deviceID, channelID); err != nil && h.assets != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(h.playTimeoutSec)*time.Second)
		defer cancel()
		if err2 := h.assets.StopLiveByChannelInternalCode(ctx, channelID); err2 == nil {
			response.OK(c, nil)
			return
		}
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *PlayHandler) Broadcast(c *gin.Context) {
	broadcastMode := c.Query("broadcastMode") == "true"
	result, err := h.svc.AudioBroadcast(c.Param("deviceId"), c.Param("channelId"), broadcastMode)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *PlayHandler) BroadcastStop(c *gin.Context) {
	if err := h.svc.StopAudioBroadcast(c.Param("deviceId"), c.Param("channelId")); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}
