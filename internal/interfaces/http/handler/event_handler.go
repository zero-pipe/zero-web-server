package handler

import (
	"context"
	"strconv"
	"time"

	eventingest "zero-web-server/internal/application/eventingest"
	domainevent "zero-web-server/internal/domain/event"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type EventHandler struct {
	svc *eventingest.Service
}

func NewEventHandler(svc *eventingest.Service) *EventHandler { return &EventHandler{svc: svc} }

func (h *EventHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "20"))
	begin, _ := strconv.ParseInt(c.Query("beginTime"), 10, 64)
	end, _ := strconv.ParseInt(c.Query("endTime"), 10, 64)
	f := domainevent.ListFilter{
		DeviceID:           c.Query("deviceId"),
		DeviceInternalCode: c.Query("deviceInternalCode"),
		EventType:          c.Query("eventType"),
		Status:             c.Query("status"),
		Begin:              begin,
		End:                end,
	}
	list, total, err := h.svc.List(c.Request.Context(), page, count, f)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"list": list, "total": total, "pageNum": page, "pageSize": count})
}

func (h *EventHandler) Get(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	e, err := h.svc.Get(ctx, c.Param("id"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, e)
}
