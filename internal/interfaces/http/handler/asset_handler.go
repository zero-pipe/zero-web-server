package handler

import (
	"strconv"

	assetapp "zero-web-server/internal/application/asset"
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
