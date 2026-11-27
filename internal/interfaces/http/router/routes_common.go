package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

func registerCommon(parent *gin.RouterGroup, commonChannelHandler *handler.CommonChannelHandler) {
	g := parent.Group("/common/channel")
	// 通道管理 CRUD / 地图 / 组织归属（无重复语义，保留）。
	g.GET("/one", commonChannelHandler.One)
	g.GET("/list", commonChannelHandler.List)
	g.POST("/add", commonChannelHandler.Add)
	g.POST("/update", commonChannelHandler.Update)
	g.POST("/reset", commonChannelHandler.Reset)
	g.GET("/industry/list", commonChannelHandler.IndustryList)
	g.GET("/type/list", commonChannelHandler.TypeList)
	g.GET("/network/identification/list", commonChannelHandler.NetworkList)
	g.GET("/map/list", commonChannelHandler.MapList)
	g.GET("/map/tile/:z/:x/:y", commonChannelHandler.MapTile)
	g.GET("/map/thin/tile/:z/:x/:y", commonChannelHandler.MapThinTile)
	g.POST("/map/reset-level", commonChannelHandler.MapResetLevel)
	g.POST("/map/thin/draw", commonChannelHandler.MapThinDraw)
	g.GET("/map/thin/clear", commonChannelHandler.MapThinClear)
	g.GET("/map/thin/save", commonChannelHandler.MapThinSave)
	g.GET("/map/thin/progress", commonChannelHandler.MapThinProgress)
	g.POST("/region/add", commonChannelHandler.RegionAdd)
	g.POST("/region/delete", commonChannelHandler.RegionDelete)
	g.POST("/region/device/add", commonChannelHandler.RegionDeviceAdd)
	g.POST("/region/device/delete", commonChannelHandler.RegionDeviceDelete)
	g.POST("/group/add", commonChannelHandler.GroupAdd)
	g.POST("/group/delete", commonChannelHandler.GroupDelete)
	g.POST("/group/device/add", commonChannelHandler.GroupDeviceAdd)
	g.POST("/group/device/delete", commonChannelHandler.GroupDeviceDelete)
	g.GET("/civilcode/list", commonChannelHandler.CivilCodeList)
	g.GET("/parent/list", commonChannelHandler.ParentList)
}
