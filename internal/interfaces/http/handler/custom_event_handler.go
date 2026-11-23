package handler

import (
	"strconv"

	customeventapp "zero-web-server/internal/application/customevent"
	domaince "zero-web-server/internal/domain/customevent"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

// CustomEventHandler 自定义事件查询（平台不理解语义，仅按基础字段与 raw 展示）。
type CustomEventHandler struct {
	svc *customeventapp.Service
}

func NewCustomEventHandler(svc *customeventapp.Service) *CustomEventHandler {
	return &CustomEventHandler{svc: svc}
}

func (h *CustomEventHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "15"))
	begin, _ := strconv.ParseInt(c.Query("begin"), 10, 64)
	end, _ := strconv.ParseInt(c.Query("end"), 10, 64)
	list, total, err := h.svc.List(c.Request.Context(), page, count, domaince.ListFilter{
		DeviceInternalCode:  c.Query("deviceCode"),
		ChannelInternalCode: c.Query("channelCode"),
		MajorType:           c.Query("majorType"),
		MinorType:           c.Query("minorType"),
		Begin:               begin,
		End:                 end,
	})
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"list": list, "total": total, "pageNum": page, "pageSize": count})
}

func (h *CustomEventHandler) Types(c *gin.Context) {
	types, err := h.svc.Types(c.Request.Context(), c.Query("deviceCode"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, types)
}

func (h *CustomEventHandler) Get(c *gin.Context) {
	e, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, e)
}

func (h *CustomEventHandler) Clear(c *gin.Context) {
	var body struct {
		BeginTime int64 `json:"beginTime"`
		EndTime   int64 `json:"endTime"`
		All       bool  `json:"all"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数无效")
		return
	}
	n, err := h.svc.ClearRange(c.Request.Context(), body.BeginTime, body.EndTime, body.All)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": n})
}
