package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	assetapp "zero-web-server/internal/application/asset"
	domainasset "zero-web-server/internal/domain/asset"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type AssetHandler struct {
	svc *assetapp.Service
}

func NewAssetHandler(svc *assetapp.Service) *AssetHandler { return &AssetHandler{svc: svc} }

func (h *AssetHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "15"))
	list, total, err := h.svc.List(page, count, c.Query("query"), c.Query("protocol"), c.Query("status"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	for _, item := range list {
		if item.Binding != nil {
			item.Binding.ConfigJSON = maskSecrets(item.Binding.ConfigJSON)
		}
	}
	response.OK(c, gin.H{"list": list, "total": total, "pageNum": page, "pageSize": count})
}

func (h *AssetHandler) Channels(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "15"))
	list, total, err := h.svc.ListChannels(page, count, c.Query("query"), c.Query("protocol"), c.Query("deviceInternalCode"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"list": list, "total": total, "pageNum": page, "pageSize": count})
}

func (h *AssetHandler) Detail(c *gin.Context) {
	detail, err := h.svc.GetDetail(c.Param("id"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	if detail.Binding != nil {
		detail.Binding.ConfigJSON = maskSecrets(detail.Binding.ConfigJSON)
	}
	response.OK(c, detail)
}

func (h *AssetHandler) Tree(c *gin.Context) {
	list, err := h.svc.Tree(c.Query("query"), c.Query("protocol"), c.Query("status"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	for _, node := range list {
		if node.Binding != nil {
			node.Binding.ConfigJSON = maskSecrets(node.Binding.ConfigJSON)
		}
	}
	response.OK(c, gin.H{"list": list, "total": len(list)})
}

func (h *AssetHandler) UpdateDevice(c *gin.Context) {
	var patch domainasset.DevicePatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误: "+err.Error())
		return
	}
	dev, err := h.svc.UpdateDevice(c.Param("id"), patch)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, dev)
}

func (h *AssetHandler) UpdateChannel(c *gin.Context) {
	var patch domainasset.ResourcePatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误: "+err.Error())
		return
	}
	res, err := h.svc.UpdateChannel(c.Param("code"), patch)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, res)
}

func (h *AssetHandler) Resources(c *gin.Context) {
	detail, err := h.svc.GetDetail(c.Param("id"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"list": detail.Resources, "total": len(detail.Resources)})
}

func (h *AssetHandler) ResourceDetail(c *gin.Context) {
	detail, err := h.svc.ResourceDetail(c.Param("id"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, detail)
}

func (h *AssetHandler) Profiles(c *gin.Context) {
	detail, err := h.svc.ResourceDetail(c.Param("id"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"list": detail.Profiles, "total": len(detail.Profiles)})
}

func (h *AssetHandler) Capabilities(c *gin.Context) {
	detail, err := h.svc.ResourceDetail(c.Param("id"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"list": detail.Capabilities, "total": len(detail.Capabilities)})
}

func (h *AssetHandler) Sync(c *gin.Context) {
	if err := h.svc.Sync(c.Request.Context(), c.Param("id")); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *AssetHandler) StartLive(c *gin.Context) {
	var body struct {
		ProfileID string `json:"profileId"`
	}
	_ = c.ShouldBindJSON(&body)
	content, err := h.svc.StartLive(c.Request.Context(), c.Param("id"), body.ProfileID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, content)
}

func (h *AssetHandler) StopLive(c *gin.Context) {
	if err := h.svc.StopLive(c.Request.Context(), c.Param("id")); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

// QueryRecords 检索设备录像。body: {startTime,endTime,streamType}，时间为
// "2006-01-02 15:04:05" 或 RFC3339 字符串；streamType 不传按"同段只返一种"处理。
func (h *AssetHandler) QueryRecords(c *gin.Context) {
	var body struct {
		StartTime  string `json:"startTime"`
		EndTime    string `json:"endTime"`
		StreamType int    `json:"streamType"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.FromError(c, fmt.Errorf("请求参数格式错误: %w", err))
		return
	}
	out, err := h.svc.QueryRecords(c.Request.Context(), c.Param("id"), body.StartTime, body.EndTime, body.StreamType)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

// StartPlayback 起一路回放流。body: {startTime,endTime,stream:main|sub}。
func (h *AssetHandler) StartPlayback(c *gin.Context) {
	var body struct {
		StartTime string `json:"startTime"`
		EndTime   string `json:"endTime"`
		Stream    string `json:"stream"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.FromError(c, fmt.Errorf("请求参数格式错误: %w", err))
		return
	}
	content, err := h.svc.StartPlayback(c.Request.Context(), c.Param("id"), body.StartTime, body.EndTime, body.Stream)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, content)
}

func (h *AssetHandler) StopPlayback(c *gin.Context) {
	if err := h.svc.StopPlayback(c.Request.Context(), c.Param("id")); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

// PTZ 统一云台控制：command=方向/变倍，verSpeed/horizonSpeed 上限 255，zoomSpeed 上限 16。
func (h *AssetHandler) PTZ(c *gin.Context) {
	command := c.Query("command")
	if command == "" {
		response.Error(c, response.CodeBadReq, "缺少 command")
		return
	}
	horizonSpeed, _ := strconv.Atoi(c.DefaultQuery("horizonSpeed", "100"))
	verticalSpeed, _ := strconv.Atoi(c.DefaultQuery("verticalSpeed", "100"))
	zoomSpeed, _ := strconv.Atoi(c.DefaultQuery("zoomSpeed", "16"))
	if err := h.svc.PTZ(c.Request.Context(), c.Param("id"), command, horizonSpeed, verticalSpeed, zoomSpeed); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

// QueryPreset 查询预置位列表。
func (h *AssetHandler) QueryPreset(c *gin.Context) {
	list, err := h.svc.QueryPresets(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, list)
}

// CallPreset 调用预置位。
func (h *AssetHandler) CallPreset(c *gin.Context) {
	if err := h.svc.CallPreset(c.Request.Context(), c.Param("id"), c.Query("presetId")); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

// AddPreset 设置预置位。
func (h *AssetHandler) AddPreset(c *gin.Context) {
	if err := h.svc.AddPreset(c.Request.Context(), c.Param("id"), c.Query("presetId")); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

// DeletePreset 删除预置位。
func (h *AssetHandler) DeletePreset(c *gin.Context) {
	if err := h.svc.DeletePreset(c.Request.Context(), c.Param("id"), c.Query("presetId")); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

// Snapshot 抓一张图，直接回 JPEG 字节（不落对象存储）。
func (h *AssetHandler) Snapshot(c *gin.Context) {
	data, err := h.svc.Snapshot(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "image/jpeg", data)
}

// Command 通用命令透传（自定义能力）：body {params}，params 原样转发给插件驱动。
func (h *AssetHandler) Command(c *gin.Context) {
	var body struct {
		Params json.RawMessage `json:"params"`
	}
	_ = c.ShouldBindJSON(&body)
	out, err := h.svc.Command(c.Request.Context(), c.Param("id"), c.Param("cmd"), body.Params)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}
