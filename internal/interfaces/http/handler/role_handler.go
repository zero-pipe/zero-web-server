package handler

import (
	"strings"

	appauth "zero-web-server/internal/application/auth"
	"zero-web-server/internal/application/rbac"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	auth *appauth.Service
}

func NewRoleHandler(auth *appauth.Service) *RoleHandler {
	return &RoleHandler{auth: auth}
}

func (h *RoleHandler) All(c *gin.Context) {
	roles, err := h.auth.ListRoles()
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, roles)
}

func (h *RoleHandler) Menus(c *gin.Context) {
	response.OK(c, rbac.MenuDefs)
}

type roleSaveReq struct {
	ID    int      `json:"id"`
	Name  string   `json:"name"`
	Menus []string `json:"menus"`
}

func (h *RoleHandler) Add(c *gin.Context) {
	var req roleSaveReq
	_ = c.ShouldBindJSON(&req)
	if strings.TrimSpace(req.Name) == "" {
		req.Name = c.Query("name")
	}
	if strings.TrimSpace(req.Name) == "" {
		response.Error(c, response.CodeBadReq, "角色名称不能为空")
		return
	}
	role, err := h.auth.AddRole(req.Name, req.Menus)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, role)
}

func (h *RoleHandler) Update(c *gin.Context) {
	var req roleSaveReq
	if err := c.ShouldBindJSON(&req); err != nil || req.ID <= 0 {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.auth.UpdateRole(req.ID, req.Name, req.Menus); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *RoleHandler) Delete(c *gin.Context) {
	id := parseIntQuery(c, "id", 0)
	if id <= 0 {
		response.Error(c, response.CodeBadReq, "id无效")
		return
	}
	if err := h.auth.DeleteRole(id); err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, nil)
}
