package handler

import (
	"context"
	"strconv"
	"time"

	playbackapp "zero-web-kit/internal/application/playback"
	"zero-web-kit/pkg/response"

	"github.com/gin-gonic/gin"
)

type PlaybackHandler struct {
	svc            *playbackapp.Service
	playTimeoutSec int
}

func NewPlaybackHandler(svc *playbackapp.Service, playTimeoutMs int) *PlaybackHandler {
	sec := playTimeoutMs / 1000
	if sec <= 0 {
		sec = 180
	}
	return &PlaybackHandler{svc: svc, playTimeoutSec: sec}
}

func (h *PlaybackHandler) Start(c *gin.Context) {
	deviceID := c.Param("deviceId")
	channelID := c.Param("channelId")
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")
	if startTime == "" || endTime == "" {
		response.Error(c, response.CodeBadReq, "startTime和endTime不能为空")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(h.playTimeoutSec)*time.Second)
	defer cancel()

	content, err := h.svc.StartPlayback(ctx, deviceID, channelID, startTime, endTime)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, content)
}

func (h *PlaybackHandler) Stop(c *gin.Context) {
	stream := c.Param("streamId")
	if err := h.svc.StopPlayback(c.Param("deviceId"), c.Param("channelId"), stream); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PlaybackHandler) Pause(c *gin.Context) {
	if err := h.svc.PausePlayback(c.Param("streamId")); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PlaybackHandler) Resume(c *gin.Context) {
	if err := h.svc.ResumePlayback(c.Param("streamId")); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PlaybackHandler) Speed(c *gin.Context) {
	speed, _ := strconv.ParseFloat(c.Param("speed"), 64)
	if speed <= 0 {
		speed = 1
	}
	if err := h.svc.SpeedPlayback(c.Param("streamId"), speed); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *PlaybackHandler) Seek(c *gin.Context) {
	seekTime, _ := strconv.ParseInt(c.Param("seekTime"), 10, 64)
	if err := h.svc.SeekPlayback(c.Param("streamId"), seekTime); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}
