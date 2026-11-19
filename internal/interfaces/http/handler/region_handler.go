package handler

import (
	"strings"
	regionapp "zero-web-server/internal/application/region"
	domaingroupregion "zero-web-server/internal/domain/groupregion"
	"zero-web-server/internal/dto"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type RegionHandler struct {
	svc *regionapp.Service
}

func NewRegionHandler(svc *regionapp.Service) *RegionHandler {
	return &RegionHandler{svc: svc}
}

func (h *RegionHandler) TreeList(c *gin.Context) {
	parent := parseOptionalIntQuery(c, "parent")
	hasChannel := parseOptionalBoolQuery(c, "hasChannel")
	list, err := h.svc.QueryForTree(parent, hasChannel)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *RegionHandler) TreeQuery(c *gin.Context) {
	page, count := parsePageCount(c)
	response.OK(c, dto.NewPageInfo([]any{}, 0, page, count))
}

func (h *RegionHandler) Add(c *gin.Context) {
	var region domaingroupregion.RegionRecord
	if err := c.ShouldBindJSON(&region); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.Add(&region); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *RegionHandler) Update(c *gin.Context) {
	var region domaingroupregion.RegionRecord
	if err := c.ShouldBindJSON(&region); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.Update(&region); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *RegionHandler) Delete(c *gin.Context) {
	id := parseOptionalIntQuery(c, "id")
	if id == nil || *id <= 0 {
		response.Error(c, response.CodeBadReq, "id 不能为空")
		return
	}
	if err := h.svc.Delete(*id); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *RegionHandler) Path(c *gin.Context) { response.OK(c, []any{}) }
// AddByCivilCode GET /api/v1/region/addByCivilCode?civilCode=530102 —— 一键建树。
func (h *RegionHandler) AddByCivilCode(c *gin.Context) {
	code := strings.TrimSpace(c.Query("civilCode"))
	if code == "" {
		response.Error(c, response.CodeBadReq, "civilCode 不能为空")
		return
	}
	created, err := h.svc.AddByCivilCode(code)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, created)
}

// CodeMappingList GET /api/v1/region/code_mapping/list
func (h *RegionHandler) CodeMappingList(c *gin.Context) {
	list, err := h.svc.ListCodeMappings()
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, list)
}

// CodeMappingSave POST /api/v1/region/code_mapping/save
func (h *RegionHandler) CodeMappingSave(c *gin.Context) {
	var body struct {
		RegionDeviceID string `json:"regionDeviceId"`
		GBCode         string `json:"gbCode"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.SaveCodeMapping(body.RegionDeviceID, body.GBCode); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

// CodeMappingDelete DELETE /api/v1/region/code_mapping/delete?regionDeviceId=
func (h *RegionHandler) CodeMappingDelete(c *gin.Context) {
	if err := h.svc.DeleteCodeMapping(c.Query("regionDeviceId")); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *RegionHandler) Description(c *gin.Context) {
	response.OK(c, h.svc.GetDescription(c.Query("civilCode")))
}

func (h *RegionHandler) BaseChildList(c *gin.Context) {
	response.OK(c, h.svc.GetAllChild(c.Query("parent")))
}
