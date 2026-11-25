package handler

import (
	"strconv"

	dictapp "zero-web-server/internal/application/dict"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

// DictHandler 数据字典维护（系统配置）+ 下拉数据源。
type DictHandler struct {
	svc *dictapp.Service
}

func NewDictHandler(svc *dictapp.Service) *DictHandler {
	return &DictHandler{svc: svc}
}

func (h *DictHandler) ListTypes(c *gin.Context) {
	list, err := h.svc.ListTypes()
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"list": list})
}

// ListItems GET /dict/items?type=channel_type&enabled=1
func (h *DictHandler) ListItems(c *gin.Context) {
	enabledOnly := c.Query("enabled") == "1" || c.Query("enabled") == "true"
	list, err := h.svc.ListItems(c.Query("type"), enabledOnly)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"list": list})
}

func (h *DictHandler) CreateType(c *gin.Context) {
	var body struct {
		Code   string `json:"code"`
		Name   string `json:"name"`
		Remark string `json:"remark"`
		Sort   int    `json:"sort"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	t, err := h.svc.CreateType(body.Code, body.Name, body.Remark, body.Sort)
	if err != nil {
		if dictapp.IsBadRequest(err) {
			response.Error(c, response.CodeBadReq, err.Error())
			return
		}
		response.FromError(c, err)
		return
	}
	response.OK(c, t)
}

func (h *DictHandler) UpdateType(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var body struct {
		Name   string `json:"name"`
		Remark string `json:"remark"`
		Sort   int    `json:"sort"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.UpdateType(id, body.Name, body.Remark, body.Sort); err != nil {
		if dictapp.IsBadRequest(err) {
			response.Error(c, response.CodeBadReq, err.Error())
			return
		}
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *DictHandler) DeleteType(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.DeleteType(id); err != nil {
		if dictapp.IsBadRequest(err) {
			response.Error(c, response.CodeBadReq, err.Error())
			return
		}
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *DictHandler) CreateItem(c *gin.Context) {
	var body struct {
		TypeCode string `json:"typeCode"`
		Value    string `json:"value"`
		Label    string `json:"label"`
		Sort     int    `json:"sort"`
		Enabled  *bool  `json:"enabled"`
		Remark   string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	it, err := h.svc.CreateItem(body.TypeCode, body.Value, body.Label, body.Sort, enabled, body.Remark)
	if err != nil {
		if dictapp.IsBadRequest(err) {
			response.Error(c, response.CodeBadReq, err.Error())
			return
		}
		response.FromError(c, err)
		return
	}
	response.OK(c, it)
}

func (h *DictHandler) UpdateItem(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var body struct {
		Label   string `json:"label"`
		Sort    int    `json:"sort"`
		Enabled *bool  `json:"enabled"`
		Remark  string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	if err := h.svc.UpdateItem(id, body.Label, body.Sort, enabled, body.Remark); err != nil {
		if dictapp.IsBadRequest(err) {
			response.Error(c, response.CodeBadReq, err.Error())
			return
		}
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *DictHandler) DeleteItem(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.DeleteItem(id); err != nil {
		if dictapp.IsBadRequest(err) {
			response.Error(c, response.CodeBadReq, err.Error())
			return
		}
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}
