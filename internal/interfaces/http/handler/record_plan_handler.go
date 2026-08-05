package handler

import (
	"strconv"

	recordplanapp "zero-web-server/internal/application/recordplan"
	"zero-web-server/internal/infrastructure/persistence/model"
	"zero-web-server/internal/interfaces/http/dto"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type RecordPlanHandler struct {
	svc *recordplanapp.Service
}

func NewRecordPlanHandler(svc *recordplanapp.Service) *RecordPlanHandler {
	return &RecordPlanHandler{svc: svc}
}

func (h *RecordPlanHandler) Add(c *gin.Context) {
	var body struct {
		Name         string                  `json:"name"`
		PlanItemList []model.RecordPlanItem  `json:"planItemList"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.Add(body.Name, body.PlanItemList); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *RecordPlanHandler) Update(c *gin.Context) {
	var body struct {
		ID           int                    `json:"id"`
		Name         string                 `json:"name"`
		PlanItemList []model.RecordPlanItem `json:"planItemList"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.Update(body.ID, body.Name, body.PlanItemList); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *RecordPlanHandler) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("planId"))
	plan, err := h.svc.Get(id)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, plan)
}

func (h *RecordPlanHandler) Query(c *gin.Context) {
	page, count := parsePageCount(c)
	list, total, err := h.svc.Query(page, count, c.Query("query"))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, dto.NewPageInfo(list, total, page, count))
}

func (h *RecordPlanHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("planId"))
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *RecordPlanHandler) Link(c *gin.Context) {
	var param recordplanapp.LinkParam
	if err := c.ShouldBindJSON(&param); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.Link(param); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *RecordPlanHandler) ChannelList(c *gin.Context) {
	page, count := parsePageCount(c)
	planID, _ := strconv.Atoi(c.Query("planId"))
	var hasLink *bool
	if v := c.Query("hasLink"); v != "" {
		b := v == "true"
		hasLink = &b
	}
	list, total, err := h.svc.ChannelList(page, count, planID, c.Query("query"), hasLink, c.Query("online"), c.Query("channelType"))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	// 前端表格用 gb* 字段；库里 gb_name 为空时回退 name，并补 gbId 兼容旧前端
	views := make([]recordChannelView, 0, len(list))
	for i := range list {
		views = append(views, newRecordChannelView(list[i]))
	}
	response.OK(c, dto.NewPageInfo(views, total, page, count))
}

type recordChannelView struct {
	model.GBDeviceChannel
	GbID int `json:"gbId"`
}

func newRecordChannelView(ch model.GBDeviceChannel) recordChannelView {
	if ch.GBName == "" {
		ch.GBName = ch.Name
	}
	if ch.GBDeviceID == "" {
		ch.GBDeviceID = ch.DeviceID
	}
	if ch.GBManufacturer == "" {
		ch.GBManufacturer = ch.Manufacturer
	}
	if ch.GBStatus == "" {
		ch.GBStatus = ch.Status
	}
	return recordChannelView{GBDeviceChannel: ch, GbID: ch.ID}
}
