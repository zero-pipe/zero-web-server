package handler

import (
	"strconv"

	alarmapp "zero-web-kit/internal/application/alarm"
	"zero-web-kit/internal/interfaces/http/dto"
	"zero-web-kit/pkg/response"

	"github.com/gin-gonic/gin"
)

type AlarmHandler struct {
	svc *alarmapp.Service
}

func NewAlarmHandler(svc *alarmapp.Service) *AlarmHandler {
	return &AlarmHandler{svc: svc}
}

func (h *AlarmHandler) List(c *gin.Context) {
	page, count := parsePageCount(c)
	var alarmType *int
	types := c.QueryArray("alarmType")
	if len(types) > 0 {
		if v, err := strconv.Atoi(types[0]); err == nil {
			alarmType = &v
		}
	}
	list, total, err := h.svc.List(page, count, alarmType, c.Query("beginTime"), c.Query("endTime"))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, dto.NewPageInfo(list, total, page, count))
}

func (h *AlarmHandler) Delete(c *gin.Context) {
	var ids []int
	if err := c.ShouldBindJSON(&ids); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.Delete(ids); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *AlarmHandler) Clear(c *gin.Context) {
	var alarmType *int
	types := c.QueryArray("alarmType")
	if len(types) > 0 {
		if v, err := strconv.Atoi(types[0]); err == nil {
			alarmType = &v
		}
	}
	if err := h.svc.Clear(alarmType, c.Query("beginTime"), c.Query("endTime")); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, 0)
}

func (h *AlarmHandler) Snap(c *gin.Context) {
	c.Status(204)
}
