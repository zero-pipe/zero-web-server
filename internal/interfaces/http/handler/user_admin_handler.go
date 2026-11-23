package handler

import (
	"zero-web-server/internal/dto"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

func (h *UserHandler) ListUsers(c *gin.Context) {
	page, count := parsePageCount(c)
	list, total, err := h.auth.ListUsers(page, count)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, dto.NewPageInfo(list, total, page, count))
}

func (h *UserHandler) AddUser(c *gin.Context) {
	username := c.Query("username")
	password := c.Query("password")
	roleID := parseIntQuery(c, "roleId", 1)
	if username == "" || password == "" {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.auth.AddUser(username, password, roleID); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := parseIntQuery(c, "id", 0)
	if id <= 0 {
		response.Error(c, response.CodeBadReq, "id无效")
		return
	}
	if err := h.auth.DeleteUser(id); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	username, _ := c.Get("username")
	name, _ := username.(string)
	oldPassword := c.Query("oldPassword")
	newPassword := c.Query("password")
	if err := h.auth.ChangePassword(name, oldPassword, newPassword); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *UserHandler) ChangePasswordForAdmin(c *gin.Context) {
	userID := parseIntQuery(c, "userId", 0)
	password := c.Query("password")
	if userID <= 0 || password == "" {
		response.Error(c, response.CodeBadReq, "参数错误")
		return
	}
	if err := h.auth.ChangePasswordForAdmin(userID, password); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}
