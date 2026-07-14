package handler

import (
	"strconv"

	downstreamapp "zero-web-kit/internal/application/downstream"
	domainsub "zero-web-kit/internal/domain/subordinate"
	"zero-web-kit/pkg/response"

	"github.com/gin-gonic/gin"
)

type SubordinateHandler struct {
	svc *downstreamapp.Service
}

func NewSubordinateHandler(svc *downstreamapp.Service) *SubordinateHandler {
	return &SubordinateHandler{svc: svc}
}

func (h *SubordinateHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "20"))
	query := c.Query("query")
	list, total, err := h.svc.List(page, count, query)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, gin.H{"list": list, "total": total})
}

func (h *SubordinateHandler) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	p, err := h.svc.Get(id)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, p)
}

func (h *SubordinateHandler) Add(c *gin.Context) {
	var body domainsub.Platform
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Add(&body); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, body)
}

func (h *SubordinateHandler) Update(c *gin.Context) {
	var body domainsub.Platform
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, response.CodeBadReq, "参数错误: "+err.Error())
		return
	}
	if id, err := strconv.Atoi(c.Param("id")); err == nil {
		body.ID = id
	}
	if err := h.svc.Update(&body); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, body)
}

func (h *SubordinateHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}
