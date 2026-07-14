package handler

import (
	"strconv"

	groupapp "zero-web-server/internal/application/group"
	"zero-web-server/internal/infrastructure/persistence"
	"zero-web-server/internal/interfaces/http/dto"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type GroupHandler struct {
	svc *groupapp.Service
}

func NewGroupHandler(svc *groupapp.Service) *GroupHandler {
	return &GroupHandler{svc: svc}
}

func (h *GroupHandler) TreeList(c *gin.Context) {
	query := c.Query("query")
	parent := parseOptionalIntQuery(c, "parent")
	hasChannel := parseOptionalBoolQuery(c, "hasChannel")
	list, err := h.svc.QueryForTree(query, parent, hasChannel)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *GroupHandler) TreeQuery(c *gin.Context) {
	page, count := parsePageCount(c)
	response.OK(c, dto.NewPageInfo([]any{}, 0, page, count))
}

func (h *GroupHandler) Add(c *gin.Context) {
	var group persistence.GroupRecord
	if err := c.ShouldBindJSON(&group); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.Add(&group); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *GroupHandler) Update(c *gin.Context) {
	var group persistence.GroupRecord
	if err := c.ShouldBindJSON(&group); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.svc.Update(&group); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *GroupHandler) Delete(c *gin.Context) {
	id, ok := parseRequiredIntQuery(c, "id")
	if !ok {
		response.Error(c, response.CodeBadReq, "分组id不存在")
		return
	}
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *GroupHandler) Path(c *gin.Context) {
	path, err := h.svc.GetPath(c.Query("deviceId"), c.Query("businessGroup"))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, path)
}

func parseRequiredIntQuery(c *gin.Context, key string) (int, bool) {
	v := c.Query(key)
	if v == "" {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return n, true
}

func parseOptionalIntQuery(c *gin.Context, key string) *int {
	v := c.Query(key)
	if v == "" {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return nil
	}
	return &n
}

func parseOptionalBoolQuery(c *gin.Context, key string) *bool {
	v := c.Query(key)
	if v == "" {
		return nil
	}
	b := v == "true" || v == "1"
	return &b
}
