package handler

import (
	"strconv"

	eventsubapp "zero-web-server/internal/application/eventsub"
	infraapp "zero-web-server/internal/application/infraconfig"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type EventSubHandler struct {
	subs  *eventsubapp.Service
	infra *infraapp.Service
}

func NewEventSubHandler(subs *eventsubapp.Service, infra *infraapp.Service) *EventSubHandler {
	return &EventSubHandler{subs: subs, infra: infra}
}

func (h *EventSubHandler) Arm(c *gin.Context) {
	var body struct {
		DeviceID   int      `json:"deviceId"`
		EventTypes []string `json:"eventTypes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.DeviceID <= 0 {
		response.Error(c, response.CodeBadReq, "deviceId 无效")
		return
	}
	sub, err := h.subs.Arm(c.Request.Context(), body.DeviceID, body.EventTypes)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, sub)
}

func (h *EventSubHandler) Disarm(c *gin.Context) {
	var body struct {
		DeviceID int `json:"deviceId"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.DeviceID <= 0 {
		response.Error(c, response.CodeBadReq, "deviceId 无效")
		return
	}
	if err := h.subs.Disarm(c.Request.Context(), body.DeviceID); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *EventSubHandler) Status(c *gin.Context) {
	deviceID, _ := strconv.Atoi(c.Query("deviceId"))
	if deviceID <= 0 {
		response.Error(c, response.CodeBadReq, "deviceId 无效")
		return
	}
	sub, err := h.subs.Get(deviceID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	pipeline := h.infra.PipelineStatus(c.Request.Context())
	response.OK(c, gin.H{"subscription": sub, "pipeline": pipeline})
}
