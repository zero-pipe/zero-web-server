package router

import (
	"github.com/gin-gonic/gin"

	"zero-web-server/internal/interfaces/http/handler"
)

// registerDevices 设备接入的唯一命名空间：/devices。
//
// 分层：
//   - /devices           统一资产门面（列表/增删改/同步，三协议一致，按 protocol 派发）
//   - /devices/gb/*      国标专属动作（目录订阅、统计、云台/拖动变倍、配置等）
//   - /devices/onvif/*   ONVIF 专属动作（发现、事件订阅、直播、云台、预置位）
//   - /devices/plugin/*  插件设备专属动作（子设备同步、布防/播放/云台）
//
// 协议专属接口复用原处理器，仅统一挂在该命名空间下；新增协议只需在此加一个子组。
func registerDevices(
	parent *gin.RouterGroup,
	daH *handler.DeviceAccessHandler,
	deviceHandler *handler.DeviceHandler,
	onvifHandler *handler.ONVIFHandler,
	pluginHandler *handler.PluginHandler,
	snapHandler *handler.SnapHandler,
) {
	g := parent.Group("/devices")

	// 统一资产门面
	if daH != nil {
		g.GET("", daH.List)
		g.GET("/detail", daH.Get)
		g.POST("", daH.Create)
		g.PUT("", daH.Update)
		g.DELETE("", daH.Delete)
		g.POST("/delete", daH.Delete)
		g.POST("/sync", daH.Sync)
	}

	registerDeviceQuery(g, deviceHandler, snapHandler) // /devices/gb/query/*
	registerDeviceControl(g, deviceHandler)            // /devices/gb/control/*
	registerDeviceConfig(g, deviceHandler)             // /devices/gb/config/*
	registerOnvif(g, onvifHandler)                     // /devices/onvif/*
	registerPluginDevices(g, pluginHandler)            // /devices/plugin/*
}
