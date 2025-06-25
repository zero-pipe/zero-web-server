package handler

import (
	"zero-web-kit/pkg/response"

	"github.com/gin-gonic/gin"
)

// JT1078Handler 部标 JT1078 接口占位实现，对齐 WVP 前端路径，避免 404。
// 完整 JT1078 协议栈尚未接入，控制类接口返回业务提示。
type JT1078Handler struct{}

func NewJT1078Handler() *JT1078Handler {
	return &JT1078Handler{}
}

func (h *JT1078Handler) emptyPage(c *gin.Context) {
	page, count := parsePageCount(c)
	response.OK(c, gin.H{
		"list":     []any{},
		"total":    int64(0),
		"pageNum":  page,
		"pageSize": count,
	})
}

func (h *JT1078Handler) ok(c *gin.Context) {
	response.OK(c, nil)
}

func (h *JT1078Handler) emptyObject(c *gin.Context) {
	response.OK(c, gin.H{})
}

func (h *JT1078Handler) notEnabled(c *gin.Context) {
	response.Error(c, response.CodeError, "部标 JT1078 服务未启用")
}

func (h *JT1078Handler) TerminalList(c *gin.Context)       { h.emptyPage(c) }
func (h *JT1078Handler) TerminalQuery(c *gin.Context)      { h.emptyObject(c) }
func (h *JT1078Handler) TerminalUpdate(c *gin.Context)     { h.ok(c) }
func (h *JT1078Handler) TerminalAdd(c *gin.Context)        { h.ok(c) }
func (h *JT1078Handler) TerminalDelete(c *gin.Context)     { h.ok(c) }
func (h *JT1078Handler) ChannelList(c *gin.Context)        { h.emptyPage(c) }
func (h *JT1078Handler) ChannelUpdate(c *gin.Context)      { h.ok(c) }
func (h *JT1078Handler) ChannelAdd(c *gin.Context)         { h.ok(c) }

func (h *JT1078Handler) LiveStart(c *gin.Context)          { h.notEnabled(c) }
func (h *JT1078Handler) LiveStop(c *gin.Context)           { h.ok(c) }
func (h *JT1078Handler) TalkStart(c *gin.Context)          { h.notEnabled(c) }
func (h *JT1078Handler) TalkStop(c *gin.Context)           { h.ok(c) }
func (h *JT1078Handler) PTZ(c *gin.Context)                { h.notEnabled(c) }
func (h *JT1078Handler) Wiper(c *gin.Context)              { h.notEnabled(c) }
func (h *JT1078Handler) FillLight(c *gin.Context)          { h.notEnabled(c) }
func (h *JT1078Handler) RecordList(c *gin.Context)         { h.emptyPage(c) }
func (h *JT1078Handler) PlaybackStart(c *gin.Context)      { h.notEnabled(c) }
func (h *JT1078Handler) PlaybackDownloadURL(c *gin.Context) { h.emptyObject(c) }
func (h *JT1078Handler) PlaybackControl(c *gin.Context)    { h.ok(c) }
func (h *JT1078Handler) PlaybackStop(c *gin.Context)       { h.ok(c) }
func (h *JT1078Handler) PlaybackDownload(c *gin.Context)   { h.notEnabled(c) }
func (h *JT1078Handler) ConfigGet(c *gin.Context)          { h.emptyObject(c) }
func (h *JT1078Handler) ConfigSet(c *gin.Context)          { h.ok(c) }
func (h *JT1078Handler) Attribute(c *gin.Context)          { h.emptyObject(c) }
func (h *JT1078Handler) LinkDetection(c *gin.Context)      { h.emptyObject(c) }
func (h *JT1078Handler) PositionInfo(c *gin.Context)       { h.emptyObject(c) }
func (h *JT1078Handler) TextMsg(c *gin.Context)            { h.ok(c) }
func (h *JT1078Handler) TelephoneCallback(c *gin.Context) { h.ok(c) }
func (h *JT1078Handler) DriverInformation(c *gin.Context)  { h.emptyObject(c) }
func (h *JT1078Handler) FactoryReset(c *gin.Context)       { h.ok(c) }
func (h *JT1078Handler) Reset(c *gin.Context)              { h.ok(c) }
func (h *JT1078Handler) Connection(c *gin.Context)         { h.ok(c) }
func (h *JT1078Handler) ControlDoor(c *gin.Context)        { h.ok(c) }
func (h *JT1078Handler) MediaAttribute(c *gin.Context)     { h.emptyObject(c) }
func (h *JT1078Handler) MediaList(c *gin.Context)          { h.emptyPage(c) }
func (h *JT1078Handler) SetPhoneBook(c *gin.Context)       { h.ok(c) }
func (h *JT1078Handler) Shooting(c *gin.Context)           { h.ok(c) }
func (h *JT1078Handler) Snap(c *gin.Context)               { h.notEnabled(c) }
func (h *JT1078Handler) MediaUpload(c *gin.Context)        { h.notEnabled(c) }
