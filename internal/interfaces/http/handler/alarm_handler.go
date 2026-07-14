package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	alarmapp "zero-web-kit/internal/application/alarm"
	objectstoreapp "zero-web-kit/internal/application/objectstore"
	"zero-web-kit/internal/interfaces/http/dto"
	"zero-web-kit/pkg/response"

	"github.com/gin-gonic/gin"
)

type AlarmHandler struct {
	svc   *alarmapp.Service
	store *objectstoreapp.Service
}

func NewAlarmHandler(svc *alarmapp.Service, store *objectstoreapp.Service) *AlarmHandler {
	return &AlarmHandler{svc: svc, store: store}
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
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Status(http.StatusNoContent)
		return
	}
	alarm, err := h.svc.Get(id)
	if err != nil || alarm == nil || strings.TrimSpace(alarm.SnapPath) == "" {
		c.Status(http.StatusNoContent)
		return
	}
	key := strings.TrimSpace(alarm.SnapPath)
	if strings.HasPrefix(key, "http://") || strings.HasPrefix(key, "https://") {
		c.Redirect(http.StatusFound, key)
		return
	}
	if h.store == nil {
		c.Status(http.StatusNoContent)
		return
	}
	if url, perr := h.store.Store().PresignGet(c.Request.Context(), key, time.Hour); perr == nil && url != "" {
		c.Redirect(http.StatusFound, url)
		return
	}
	rc, gerr := h.store.Store().Get(c.Request.Context(), key)
	if gerr != nil {
		c.Status(http.StatusNoContent)
		return
	}
	defer rc.Close()
	c.Header("Content-Type", "image/jpeg")
	_, _ = io.Copy(c.Writer, rc)
}
