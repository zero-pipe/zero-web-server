package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerUser(parent *gin.RouterGroup, userHandler *handler.UserHandler) {
	g := parent.Group("/user")
	g.GET("/login", userHandler.Login)
	g.POST("/login", userHandler.Login)
	g.GET("/logout", userHandler.Logout)
}
