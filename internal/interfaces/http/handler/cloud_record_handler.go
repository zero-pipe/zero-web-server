package handler

import (
	"context"
	"strconv"
	"time"

	cloudrecordapp "zero-web-kit/internal/application/cloudrecord"
	"zero-web-kit/internal/interfaces/http/dto"
	"zero-web-kit/pkg/response"

	"github.com/gin-gonic/gin"
)

type CloudRecordHandler struct {
	svc            *cloudrecordapp.Service
	playTimeoutSec int
}

func NewCloudRecordHandler(svc *cloudrecordapp.Service, playTimeoutMs int) *CloudRecordHandler {
	sec := playTimeoutMs / 1000
	if sec <= 0 {
		sec = 180
	}
	return &CloudRecordHandler{svc: svc, playTimeoutSec: sec}
}

// parseTimeQueryMs 支持毫秒时间戳或 "yyyy-MM-dd HH:mm:ss" / "yyyy-MM-dd"。
func parseTimeQueryMs(raw string) int64 {
	if raw == "" {
		return 0
	}
	if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return n
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return t.UnixMilli()
		}
	}
	return 0
}

func (h *CloudRecordHandler) List(c *gin.Context) {
	page, count := parsePageCount(c)
	startTime := parseTimeQueryMs(c.Query("startTime"))
	endTime := parseTimeQueryMs(c.Query("endTime"))
	asc := c.Query("ascOrder") == "true"
	list, total, err := h.svc.List(page, count, c.Query("app"), c.Query("stream"), c.Query("query"),
		c.Query("callId"), c.Query("mediaServerId"), startTime, endTime, asc)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, dto.NewPageInfo(list, total, page, count))
}

func (h *CloudRecordHandler) DateList(c *gin.Context) {
	year, _ := strconv.Atoi(c.Query("year"))
	month, _ := strconv.Atoi(c.Query("month"))
	dates, err := h.svc.DateList(c.Query("app"), c.Query("stream"), c.Query("mediaServerId"), year, month)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, dates)
}

func (h *CloudRecordHandler) Delete(c *gin.Context) {
	var body struct {
		IDs []int `json:"ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.IDs) == 0 {
		response.Error(c, response.CodeBadReq, "ids无效")
		return
	}
	if err := h.svc.Delete(body.IDs); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CloudRecordHandler) PlayPath(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("recordId"))
	info, err := h.svc.GetPlayPath(id)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, info)
}

func (h *CloudRecordHandler) LoadRecord(c *gin.Context) {
	cloudID, _ := strconv.Atoi(c.Query("cloudRecordId"))
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(h.playTimeoutSec)*time.Second)
	defer cancel()
	content, err := h.svc.LoadRecord(ctx, c.Query("app"), c.Query("stream"), cloudID)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, content)
}

func (h *CloudRecordHandler) Seek(c *gin.Context) {
	seek, _ := strconv.ParseFloat(c.Query("seek"), 64)
	if err := h.svc.Seek(c.Query("app"), c.Query("stream"), c.Query("mediaServerId"), seek, c.Query("schema")); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CloudRecordHandler) Speed(c *gin.Context) {
	speed, _ := strconv.Atoi(c.Query("speed"))
	if err := h.svc.Speed(c.Query("app"), c.Query("stream"), c.Query("mediaServerId"), speed, c.Query("schema")); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CloudRecordHandler) AddTask(c *gin.Context) {
	taskID, err := h.svc.AddTask(c.Query("app"), c.Query("stream"), c.Query("mediaServerId"), c.Query("startTime"), c.Query("endTime"))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, taskID)
}

func (h *CloudRecordHandler) TaskList(c *gin.Context) {
	var isEnd *bool
	if v := c.Query("isEnd"); v != "" {
		b := v == "true"
		isEnd = &b
	}
	list, err := h.svc.QueryTaskList(c.Query("mediaServerId"), isEnd)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, list)
}
