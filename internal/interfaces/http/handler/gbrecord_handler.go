package handler

import (
	"context"
	"time"

	playbackapp "zero-web-kit/internal/application/playback"
	sipinfra "zero-web-kit/internal/infrastructure/sip"
	"zero-web-kit/pkg/response"

	"github.com/gin-gonic/gin"
)

type GBRecordHandler struct {
	svc           *playbackapp.Service
	recordTimeout time.Duration
}

func NewGBRecordHandler(svc *playbackapp.Service, recordTimeoutMs int) *GBRecordHandler {
	if recordTimeoutMs <= 0 {
		recordTimeoutMs = 30000
	}
	return &GBRecordHandler{
		svc:           svc,
		recordTimeout: time.Duration(recordTimeoutMs) * time.Millisecond,
	}
}

func (h *GBRecordHandler) Query(c *gin.Context) {
	deviceID := c.Param("deviceId")
	channelID := c.Param("channelId")
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")
	if startTime == "" || endTime == "" {
		response.Error(c, response.CodeBadReq, "startTime和endTime不能为空")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.recordTimeout)
	defer cancel()

	info, err := h.svc.QueryRecord(ctx, deviceID, channelID, startTime, endTime)
	if err != nil {
		if err == sipinfra.ErrRecordTimeout {
			response.Error(c, response.CodeError, "录像查询超时")
			return
		}
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, info)
}

func (h *GBRecordHandler) DownloadStart(c *gin.Context) {
	deviceID := c.Param("deviceId")
	channelID := c.Param("channelId")
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")
	if startTime == "" || endTime == "" {
		response.Error(c, response.CodeBadReq, "startTime和endTime不能为空")
		return
	}
	speed := playbackapp.ParseDownloadSpeed(c.Query("downloadSpeed"))

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	content, err := h.svc.StartDownload(ctx, deviceID, channelID, startTime, endTime, speed)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, content)
}

func (h *GBRecordHandler) DownloadStop(c *gin.Context) {
	stream := c.Param("stream")
	if err := h.svc.StopDownload(c.Param("deviceId"), c.Param("channelId"), stream); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *GBRecordHandler) DownloadProgress(c *gin.Context) {
	progress, err := h.svc.DownloadProgress(c.Param("stream"))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, progress)
}
