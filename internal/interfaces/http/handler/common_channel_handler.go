package handler

import (
	"strconv"

	channelapp "zero-web-kit/internal/application/channel"
	"zero-web-kit/internal/interfaces/http/dto"
	"zero-web-kit/pkg/response"

	"github.com/gin-gonic/gin"
)

type CommonChannelHandler struct {
	svc *channelapp.Service
}

func NewCommonChannelHandler(svc *channelapp.Service) *CommonChannelHandler {
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
	var body channelapp.View
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误: "+err.Error())
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
	// GB/T 28181 行业编码，按 code 升序
	response.OK(c, []gin.H{
		{"code": "00", "name": "社会治安"},
		{"code": "01", "name": "交通管理"},
		{"code": "02", "name": "刑侦"},
		{"code": "03", "name": "禁毒"},
		{"code": "04", "name": "安保"},
		{"code": "05", "name": "出入境"},
		{"code": "06", "name": "网安"},
		{"code": "07", "name": "其他警务"},
		{"code": "08", "name": "防火"},
		{"code": "09", "name": "其他"},
		{"code": "10", "name": "司法行政"},
		{"code": "11", "name": "纪检监察"},
		{"code": "12", "name": "安全防范"},
		{"code": "13", "name": "检察"},
		{"code": "14", "name": "法院"},
		{"code": "15", "name": "海关"},
		{"code": "16", "name": "出入境检验检疫"},
		{"code": "20", "name": "教育"},
		{"code": "21", "name": "卫生"},
		{"code": "22", "name": "民政"},
		{"code": "23", "name": "文旅"},
		{"code": "24", "name": "环保"},
		{"code": "25", "name": "安监"},
		{"code": "26", "name": "金融"},
		{"code": "27", "name": "住建"},
		{"code": "28", "name": "水利"},
		{"code": "29", "name": "农业"},
		{"code": "30", "name": "商务"},
		{"code": "31", "name": "国土资源"},
		{"code": "32", "name": "信息产业"},
		{"code": "33", "name": "质量监督检验检疫"},
		{"code": "34", "name": "新闻出版"},
		{"code": "35", "name": "食品药品监督"},
		{"code": "36", "name": "气象"},
		{"code": "37", "name": "地震"},
		{"code": "38", "name": "测绘"},
		{"code": "39", "name": "烟草专卖"},
		{"code": "40", "name": "邮政"},
		{"code": "41", "name": "社保"},
		{"code": "42", "name": "广电"},
		{"code": "43", "name": "铁路"},
		{"code": "44", "name": "交通"},
		{"code": "45", "name": "民航"},
		{"code": "46", "name": "航运"},
		{"code": "47", "name": "林业"},
		{"code": "80", "name": "社区/村镇"},
		{"code": "81", "name": "商企"},
		{"code": "82", "name": "机关事业单位"},
		{"code": "99", "name": "其他行业"},
	})
}
func (h *CommonChannelHandler) TypeList(c *gin.Context) {
	// GB/T 28181 类型编码，按 code 升序
	response.OK(c, []gin.H{
		{"code": "111", "name": "DVR"},
		{"code": "112", "name": "视频服务器"},
		{"code": "113", "name": "编码器"},
		{"code": "114", "name": "解码器"},
		{"code": "115", "name": "视频切换矩阵"},
		{"code": "116", "name": "音频切换矩阵"},
		{"code": "117", "name": "报警控制器"},
		{"code": "118", "name": "NVR"},
		{"code": "130", "name": "摄像机扩展"},
		{"code": "131", "name": "摄像机"},
		{"code": "132", "name": "网络摄像机"},
		{"code": "200", "name": "中心信令控制服务器"},
		{"code": "201", "name": "Web应用服务器"},
		{"code": "202", "name": "媒体服务器"},
		{"code": "203", "name": "代理服务器"},
		{"code": "204", "name": "安全服务器"},
		{"code": "205", "name": "报警服务器"},
		{"code": "206", "name": "数据库服务器"},
		{"code": "207", "name": "GIS服务器"},
		{"code": "208", "name": "管理服务器"},
		{"code": "209", "name": "接入网关"},
		{"code": "210", "name": "媒体存储服务器"},
		{"code": "211", "name": "信令安全路由网关"},
		{"code": "212", "name": "业务分组扩展"},
		{"code": "215", "name": "业务分组"},
		{"code": "216", "name": "虚拟组织"},
		{"code": "500", "name": "视频画面分割器"},
	})
}
func (h *CommonChannelHandler) NetworkList(c *gin.Context) {
	// GB/T 28181 网络标识，按 code 升序
	response.OK(c, []gin.H{
		{"code": "0", "name": "监控报警专网"},
		{"code": "1", "name": "公安信息网"},
		{"code": "2", "name": "政务网"},
		{"code": "3", "name": "Internet"},
		{"code": "4", "name": "社会资源接入网"},
		{"code": "5", "name": "其他"},
	})
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
	var list []channelapp.View
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

type channelToRegionByDeviceParam struct {
	CivilCode string `json:"civilCode"`
	DeviceIDs []int  `json:"deviceIds"`
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

func (h *CommonChannelHandler) RegionDeviceAdd(c *gin.Context) {
	var body channelToRegionByDeviceParam
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.AddToRegionByDevices(body.CivilCode, body.DeviceIDs); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CommonChannelHandler) RegionDeviceDelete(c *gin.Context) {
	var body channelToRegionByDeviceParam
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.DeleteFromRegionByDevices(body.DeviceIDs); err != nil {
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

type channelToGroupByDeviceParam struct {
	ParentID      string `json:"parentId"`
	BusinessGroup string `json:"businessGroup"`
	DeviceIDs     []int  `json:"deviceIds"`
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

func (h *CommonChannelHandler) GroupDeviceAdd(c *gin.Context) {
	var body channelToGroupByDeviceParam
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.AddToGroupByDevices(body.ParentID, body.BusinessGroup, body.DeviceIDs); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CommonChannelHandler) GroupDeviceDelete(c *gin.Context) {
	var body channelToGroupByDeviceParam
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.DeleteFromGroupByDevices(body.DeviceIDs); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *CommonChannelHandler) StubOK(c *gin.Context) { response.OK(c, nil) }
func (h *CommonChannelHandler) StubList(c *gin.Context) {
	response.OK(c, dto.NewPageInfo([]any{}, 0, 1, 15))
}
