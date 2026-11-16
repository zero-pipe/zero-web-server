package handler

import (
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

func (h *RegionHandler) Delete(c *gin.Context)         { response.OK(c, nil) }
func (h *RegionHandler) Path(c *gin.Context)           { response.OK(c, []any{}) }
func (h *RegionHandler) AddByCivilCode(c *gin.Context) { response.OK(c, nil) }

func (h *RegionHandler) Description(c *gin.Context) {
	response.OK(c, h.svc.GetDescription(c.Query("civilCode")))
}

func (h *RegionHandler) BaseChildList(c *gin.Context) {
	response.OK(c, h.svc.GetAllChild(c.Query("parent")))
}
