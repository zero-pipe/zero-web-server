package handler

import (
	"strconv"

	commonchannelapp "zero-web-kit/internal/application/commonchannel"
	"zero-web-kit/internal/interfaces/http/dto"
	"zero-web-kit/pkg/response"

	"github.com/gin-gonic/gin"
)

type CommonChannelHandler struct {
	svc *commonchannelapp.Service
}

func NewCommonChannelHandler(svc *commonchannelapp.Service) *CommonChannelHandler {
	return &CommonChannelHandler{svc: svc}
}

func (h *CommonChannelHandler) One(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("id"))
	v, err := h.svc.GetOne(id)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, v)
}

func (h *CommonChannelHandler) List(c *gin.Context) {
	page, count := parsePageCount(c)
	var channelType *int
	if v := c.Query("channelType"); v != "" {
		n, _ := strconv.Atoi(v)
		channelType = &n
	}
	var online, hasRecordPlan *bool
	if v := c.Query("online"); v != "" {
		b := v == "true"
		online = &b
	}
	if v := c.Query("hasRecordPlan"); v != "" {
		b := v == "true"
		hasRecordPlan = &b
	}
	list, total, err := h.svc.ListFiltered(page, count, c.Query("query"), channelType, online, hasRecordPlan, c.Query("civilCode"), c.Query("parentDeviceId"))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, dto.NewPageInfo(list, total, page, count))
}

func (h *CommonChannelHandler) Update(c *gin.Context) {
	var body commonchannelapp.View
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.Update(&body); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CommonChannelHandler) Add(c *gin.Context)    { h.Update(c) }
func (h *CommonChannelHandler) Reset(c *gin.Context)  { response.OK(c, nil) }

func (h *CommonChannelHandler) IndustryList(c *gin.Context) {
	response.OK(c, []gin.H{{"code": "00", "name": "社会治安"}})
}
func (h *CommonChannelHandler) TypeList(c *gin.Context) {
	response.OK(c, []gin.H{{"code": "131", "name": "摄像机"}})
}
func (h *CommonChannelHandler) NetworkList(c *gin.Context) {
	response.OK(c, []gin.H{{"code": "0", "name": "监控报警专网"}})
}

func (h *CommonChannelHandler) Play(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("channelId"))
	content, err := h.svc.Play(id)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, content)
}

func (h *CommonChannelHandler) PlayStop(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("channelId"))
	if err := h.svc.StopPlay(id); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CommonChannelHandler) TalkStart(c *gin.Context)   { h.broadcast(c, false) }
func (h *CommonChannelHandler) TalkStop(c *gin.Context)    { h.broadcastStop(c) }
func (h *CommonChannelHandler) BroadcastStart(c *gin.Context) { h.broadcast(c, true) }
func (h *CommonChannelHandler) BroadcastStop(c *gin.Context)  { h.broadcastStop(c) }

func (h *CommonChannelHandler) broadcast(c *gin.Context, mode bool) {
	id, _ := strconv.Atoi(c.Query("channelId"))
	result, err := h.svc.Broadcast(id, mode)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *CommonChannelHandler) broadcastStop(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("channelId"))
	if err := h.svc.BroadcastStop(id); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CommonChannelHandler) PTZ(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("channelId"))
	if err := h.svc.PTZ(id, c.Query("command"),
		parseIntQuery(c, "panSpeed", 50),
		parseIntQuery(c, "tiltSpeed", 50),
		parseIntQuery(c, "zoomSpeed", 50),
	); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CommonChannelHandler) QueryPreset(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("channelId"))
	list, err := h.svc.QueryPreset(c.Request.Context(), id)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *CommonChannelHandler) AddPreset(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("channelId"))
	presetID := c.Query("presetId")
	if presetID == "" {
		response.Error(c, response.CodeBadReq, "presetId 不能为空")
		return
	}
	if err := h.svc.AddPreset(c.Request.Context(), id, presetID, c.Query("presetName")); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CommonChannelHandler) CallPreset(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("channelId"))
	presetID := c.Query("presetId")
	if presetID == "" {
		response.Error(c, response.CodeBadReq, "presetId 不能为空")
		return
	}
	if err := h.svc.CallPreset(c.Request.Context(), id, presetID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CommonChannelHandler) DeletePreset(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("channelId"))
	presetID := c.Query("presetId")
	if presetID == "" {
		response.Error(c, response.CodeBadReq, "presetId 不能为空")
		return
	}
	if err := h.svc.DeletePreset(c.Request.Context(), id, presetID); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CommonChannelHandler) PlaybackQuery(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("channelId"))
	data, err := h.svc.PlaybackQuery(id, c.Query("startTime"), c.Query("endTime"))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, data)
}

func (h *CommonChannelHandler) Playback(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("channelId"))
	content, err := h.svc.PlaybackStart(id, c.Query("startTime"), c.Query("endTime"))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, content)
}

func (h *CommonChannelHandler) PlaybackStop(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("channelId"))
	if err := h.svc.PlaybackStop(id, c.Query("stream")); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CommonChannelHandler) PlaybackPause(c *gin.Context) {
	if err := h.svc.PlaybackPause(c.Query("stream")); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CommonChannelHandler) PlaybackResume(c *gin.Context) {
	if err := h.svc.PlaybackResume(c.Query("stream")); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CommonChannelHandler) PlaybackSpeed(c *gin.Context) {
	speed, _ := strconv.ParseFloat(c.Query("speed"), 64)
	if err := h.svc.PlaybackSpeed(c.Query("stream"), speed); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CommonChannelHandler) PlaybackSeek(c *gin.Context) {
	seekTime, _ := strconv.ParseInt(c.Query("seekTime"), 10, 64)
	if err := h.svc.PlaybackSeek(c.Query("stream"), seekTime); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CommonChannelHandler) MapList(c *gin.Context) {
	var channelType *int
	if v := c.Query("channelType"); v != "" {
		n, _ := strconv.Atoi(v)
		channelType = &n
	}
	list, err := h.svc.MapList(c.Query("query"), channelType, nil, nil)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *CommonChannelHandler) MapTile(c *gin.Context) {
	c.Data(200, "application/x-protobuf", []byte{})
}

func (h *CommonChannelHandler) MapThinTile(c *gin.Context) {
	c.Data(200, "application/x-protobuf", []byte{})
}

func (h *CommonChannelHandler) MapResetLevel(c *gin.Context) {
	response.OK(c, nil)
}

func (h *CommonChannelHandler) MapThinDraw(c *gin.Context) {
	response.OK(c, "")
}

func (h *CommonChannelHandler) MapThinClear(c *gin.Context) {
	response.OK(c, nil)
}

func (h *CommonChannelHandler) MapThinSave(c *gin.Context) {
	response.OK(c, nil)
}

func (h *CommonChannelHandler) MapThinProgress(c *gin.Context) {
	response.OK(c, gin.H{"total": 0, "current": 0, "percent": 100})
}

func (h *CommonChannelHandler) CivilCodeList(c *gin.Context) {
	h.listByAssociation(c, true)
}

func (h *CommonChannelHandler) ParentList(c *gin.Context) {
	h.listByAssociation(c, false)
}

func (h *CommonChannelHandler) listByAssociation(c *gin.Context, byCivilCode bool) {
	page, count := parsePageCount(c)
	var channelType *int
	if v := c.Query("channelType"); v != "" {
		n, _ := strconv.Atoi(v)
		channelType = &n
	}
	var online *bool
	if v := c.Query("online"); v != "" {
		b := v == "true"
		online = &b
	}
	query := c.Query("query")
	var list []commonchannelapp.View
	var total int64
	var err error
	if byCivilCode {
		list, total, err = h.svc.ListByCivilCode(page, count, query, channelType, online, c.Query("civilCode"))
	} else {
		list, total, err = h.svc.ListByGroupParent(page, count, query, channelType, online, c.Query("groupDeviceId"))
	}
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, dto.NewPageInfo(list, total, page, count))
}

type channelToRegionParam struct {
	CivilCode  string `json:"civilCode"`
	ChannelIDs []int  `json:"channelIds"`
}

func (h *CommonChannelHandler) RegionAdd(c *gin.Context) {
	var body channelToRegionParam
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.AddToRegion(body.CivilCode, body.ChannelIDs); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CommonChannelHandler) RegionDelete(c *gin.Context) {
	var body channelToRegionParam
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.DeleteFromRegion(body.ChannelIDs); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

type channelToGroupParam struct {
	ParentID      string `json:"parentId"`
	BusinessGroup string `json:"businessGroup"`
	ChannelIDs    []int  `json:"channelIds"`
}

func (h *CommonChannelHandler) GroupAdd(c *gin.Context) {
	var body channelToGroupParam
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.AddToGroup(body.ParentID, body.BusinessGroup, body.ChannelIDs); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CommonChannelHandler) GroupDelete(c *gin.Context) {
	var body channelToGroupParam
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.DeleteFromGroup(body.ChannelIDs); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CommonChannelHandler) StubOK(c *gin.Context) { response.OK(c, nil) }
func (h *CommonChannelHandler) StubList(c *gin.Context) {
	response.OK(c, dto.NewPageInfo([]any{}, 0, 1, 15))
}
