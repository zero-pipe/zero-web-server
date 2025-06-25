package handler

import (
	"strconv"
	"time"

	positionapp "zero-web-kit/internal/application/position"
	"zero-web-kit/pkg/response"

	"github.com/gin-gonic/gin"
)

type PositionHandler struct {
	svc *positionapp.Service
}

func NewPositionHandler(svc *positionapp.Service) *PositionHandler {
	return &PositionHandler{svc: svc}
}

func (h *PositionHandler) History(c *gin.Context) {
	start := parsePositionTime(c.Query("start"))
	end := parsePositionTime(c.Query("end"))

	if channelID, err := strconv.Atoi(c.Query("channelId")); err == nil && channelID > 0 {
		list, err := h.svc.HistoryByChannelDBID(channelID, start, end)
		if err != nil {
			response.Error(c, response.CodeError, err.Error())
			return
		}
		response.OK(c, list)
		return
	}

	deviceID := c.Param("deviceId")
	channelGBID := c.Query("channelId")
	if channelGBID == "" {
		channelGBID = deviceID
	}

	list, err := h.svc.History(deviceID, channelGBID, start, end)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *PositionHandler) Latest(c *gin.Context) {
	channelID, _ := strconv.Atoi(c.Query("channelId"))
	if channelID <= 0 {
		response.Error(c, response.CodeBadReq, "channelId无效")
		return
	}
	pos, err := h.svc.Latest(channelID)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, pos)
}

func parsePositionTime(v string) int64 {
	if v == "" {
		return 0
	}
	if ms, err := strconv.ParseInt(v, 10, 64); err == nil {
		return ms
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", v, time.Local)
	if err != nil {
		return 0
	}
	return t.UnixMilli()
}
