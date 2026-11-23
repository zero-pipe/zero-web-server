package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
	"zero-web-server/internal/interfaces/http/middleware"
)

func registerUserMgmt(parent *gin.RouterGroup, userHandler *handler.UserHandler, roleHandler *handler.RoleHandler) {
	g := parent.Group("")
	g.Use(middleware.RequireMenu("user"))
	g.GET("/user/users", userHandler.ListUsers)
	g.POST("/user/add", userHandler.AddUser)
	g.DELETE("/user/delete", userHandler.DeleteUser)
	g.POST("/user/changePasswordForAdmin", userHandler.ChangePasswordForAdmin)

	g.GET("/role/all", roleHandler.All)
	g.GET("/role/menus", roleHandler.Menus)
	g.POST("/role/add", roleHandler.Add)
	g.POST("/role/update", roleHandler.Update)
	g.DELETE("/role/delete", roleHandler.Delete)
}
