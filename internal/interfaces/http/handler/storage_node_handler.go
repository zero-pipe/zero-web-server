package handler

import (
	storagenodeapp "zero-web-server/internal/application/storagenode"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

// StorageNodeHandler 存储节点相关接口（供录像计划「录制目标」下拉使用）。
type StorageNodeHandler struct {
	svc *storagenodeapp.Service
}

func NewStorageNodeHandler(svc *storagenodeapp.Service) *StorageNodeHandler {
	return &StorageNodeHandler{svc: svc}
}

// List 返回全部存储节点（含在线/容量），供前端选择录制目标。
func (h *StorageNodeHandler) List(c *gin.Context) {
	if h.svc == nil {
		response.OK(c, []any{})
		return
	}
	list, err := h.svc.ListView()
	if err != nil {
		response.FromError(c, err)
		return
	}
	if list == nil {
		list = []storagenodeapp.NodeView{}
	}
	response.OK(c, list)
}
