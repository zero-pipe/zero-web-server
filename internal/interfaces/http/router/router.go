package router

import (
	appauth "zero-web-kit/internal/application/auth"
	alarmapp "zero-web-kit/internal/application/alarm"
	cloudrecordapp "zero-web-kit/internal/application/cloudrecord"
	channelapp "zero-web-kit/internal/application/channel"
	deviceapp "zero-web-kit/internal/application/device"
	deviceaccess "zero-web-kit/internal/application/deviceaccess"
	groupapp "zero-web-kit/internal/application/group"
	publishauth "zero-web-kit/internal/application/publishauth"
	mediaserverapp "zero-web-kit/internal/application/mediaserver"
	onvifapp "zero-web-kit/internal/application/onvif"
	"zero-web-kit/internal/application/ops"
	upstreamapp "zero-web-kit/internal/application/upstream"
	positionapp "zero-web-kit/internal/application/position"
	playapp "zero-web-kit/internal/application/play"
	playbackapp "zero-web-kit/internal/application/playback"
	ptzapp "zero-web-kit/internal/application/ptz"
	regionapp "zero-web-kit/internal/application/region"
	recordplanapp "zero-web-kit/internal/application/recordplan"
	streampushapp "zero-web-kit/internal/application/streampush"
	streamproxyapp "zero-web-kit/internal/application/streamproxy"
	gbsipconfig "zero-web-kit/internal/application/gbsipconfig"
	objectstoreapp "zero-web-kit/internal/application/objectstore"
	snapapp "zero-web-kit/internal/application/snap"
	downstreamapp "zero-web-kit/internal/application/downstream"
	"zero-web-kit/internal/infrastructure/config"
	"zero-web-kit/internal/infrastructure/persistence"
	"zero-web-kit/internal/interfaces/hook"
	"zero-web-kit/internal/interfaces/http/handler"
	"zero-web-kit/internal/interfaces/http/middleware"
	jwtmgr "zero-web-kit/pkg/jwt"

	"github.com/gin-gonic/gin"
)

type Deps struct {
	AuthService         *appauth.Service
	ONVIFService        *onvifapp.Service
	DeviceService       *deviceapp.Service
	DeviceAccessService *deviceaccess.Service
	PlayService         *playapp.Service
	PlaybackService     *playbackapp.Service
	PTZService          *ptzapp.Service
	AlarmService        *alarmapp.Service
	PlatformService     *upstreamapp.Service
	PlatformChannelSvc  *upstreamapp.ChannelService
	SubordinateService  *downstreamapp.Service
	PositionService     *positionapp.Service
	CloudRecordService  *cloudrecordapp.Service
	StreamPushService   *streampushapp.Service
	StreamProxyService  *streamproxyapp.Service
	RecordPlanService   *recordplanapp.Service
	MediaServerService  *mediaserverapp.Service
	CommonChannelSvc    *channelapp.Service
	GroupService        *groupapp.Service
	RegionService       *regionapp.Service
	UserRepo            *persistence.UserRepository
	PublishAuth         *publishauth.PublishAuth
	StreamOnDemand      bool
	MediaBaseURL        string
	JWT                 *jwtmgr.Manager
	ServerID            string
	Version             string
	PlayTimeoutMs       int
	RecordInfoTimeoutMs int
	SIPConfig           config.SIPConfig
	GbSipConfigService  *gbsipconfig.Service
	ObjectStoreService  *objectstoreapp.Service
	SnapService         *snapapp.Service
	ServerPort          int
	MediaIP             string
	LogDir              string
	Metrics             *ops.Metrics
	Dashboard           *ops.Dashboard
}

func Setup(r *gin.Engine, deps Deps) {
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	if deps.MediaBaseURL != "" {
		webrtcProxy := handler.NewWebRTCProxyHandler(deps.MediaBaseURL)
		r.Any("/index/api/webrtc", webrtcProxy.Proxy)
		r.Any("/index/api/whep", webrtcProxy.Proxy)
		r.Any("/index/api/whep/*path", webrtcProxy.Proxy)
	}

	userHandler := handler.NewUserHandler(deps.AuthService)
	healthHandler := handler.NewHealthHandler(deps.Version)
	serverHandler := handler.NewServerHandler(deps.ServerID, deps.Metrics)
	onvifHandler := handler.NewONVIFHandler(deps.ONVIFService)
	deviceHandler := handler.NewDeviceHandler(deps.DeviceService)
	deviceAccessHandler := handler.NewDeviceAccessHandler(deps.DeviceAccessService)
	playHandler := handler.NewPlayHandler(deps.PlayService, deps.PlayTimeoutMs)
	playbackHandler := handler.NewPlaybackHandler(deps.PlaybackService, deps.PlayTimeoutMs)
	gbRecordHandler := handler.NewGBRecordHandler(deps.PlaybackService, deps.RecordInfoTimeoutMs)
	ptzHandler := handler.NewPTZHandler(deps.PTZService)
	alarmHandler := handler.NewAlarmHandler(deps.AlarmService, deps.ObjectStoreService)
	platformHandler := handler.NewPlatformHandler(deps.PlatformService, deps.PlatformChannelSvc)
	subordinateHandler := handler.NewSubordinateHandler(deps.SubordinateService)
	positionHandler := handler.NewPositionHandler(deps.PositionService)
	cloudRecordHandler := handler.NewCloudRecordHandler(deps.CloudRecordService, deps.PlayTimeoutMs)
	streamPushHandler := handler.NewStreamPushHandler(deps.StreamPushService, deps.PlayTimeoutMs)
	streamProxyHandler := handler.NewStreamProxyHandler(deps.StreamProxyService, deps.PlayTimeoutMs)
	recordPlanHandler := handler.NewRecordPlanHandler(deps.RecordPlanService)
	mediaServerHandler := handler.NewMediaServerHandler(
		deps.MediaServerService,
		deps.Dashboard,
		deps.GbSipConfigService,
		deps.SIPConfig,
		deps.MediaIP,
		deps.ServerPort,
		deps.ServerID,
		deps.Version,
	)
	gbSipConfigHandler := handler.NewGbSipConfigHandler(deps.GbSipConfigService)
	objectStoreHandler := handler.NewObjectStoreHandler(deps.ObjectStoreService)
	snapHandler := handler.NewSnapHandler(deps.SnapService)
	jt1078Handler := handler.NewJT1078Handler()
	commonChannelHandler := handler.NewCommonChannelHandler(deps.CommonChannelSvc)
	groupHandler := handler.NewGroupHandler(deps.GroupService)
	regionHandler := handler.NewRegionHandler(deps.RegionService)
	roleHandler := handler.NewRoleHandler(deps.AuthService)
	logHandler := handler.NewLogHandler(ops.NewLogService(deps.LogDir))
	mediaHook := hook.NewHandler(
		deps.PlayService, deps.PlaybackService, deps.CloudRecordService,
		deps.StreamPushService, deps.StreamProxyService, deps.RecordPlanService,
		deps.PublishAuth, deps.StreamOnDemand,
	)

	r.GET("/health", healthHandler.Health)
	// 实时日志 WebSocket（/channel/log；token 走子协议）
	r.GET("/channel/log", handler.LogChannel)
	api := r.Group("/api")
	{
		user := api.Group("/user")
		{
			user.GET("/login", userHandler.Login)
			user.POST("/login", userHandler.Login)
			user.GET("/logout", userHandler.Logout)
		}
		server := api.Group("/server")
		{
			server.GET("/version", healthHandler.Version)
			server.GET("/config", serverHandler.Config)
			server.GET("/system/info", serverHandler.SystemInfo)
		}
	}

	hookGroup := r.Group("/index/hook")
	mediaHook.Register(hookGroup)

	auth := r.Group("/api")
	auth.Use(middleware.Auth(deps.JWT, deps.UserRepo))
	{
		auth.POST("/user/userInfo", userHandler.UserInfo)
		auth.POST("/user/changePassword", userHandler.ChangePassword)

		userMgmt := auth.Group("")
		userMgmt.Use(middleware.RequireMenu("user"))
		{
			userMgmt.GET("/user/users", userHandler.ListUsers)
			userMgmt.POST("/user/add", userHandler.AddUser)
			userMgmt.DELETE("/user/delete", userHandler.DeleteUser)
			userMgmt.POST("/user/changePasswordForAdmin", userHandler.ChangePasswordForAdmin)
			userMgmt.POST("/user/changePushKey", userHandler.ChangePushKey)

			userMgmt.GET("/role/all", roleHandler.All)
			userMgmt.GET("/role/menus", roleHandler.Menus)
			userMgmt.POST("/role/add", roleHandler.Add)
			userMgmt.POST("/role/update", roleHandler.Update)
			userMgmt.DELETE("/role/delete", roleHandler.Delete)
		}

		serverAPI := auth.Group("/server")
		{
			serverAPI.GET("/media_server/list", mediaServerHandler.List)
			serverAPI.GET("/media_server/online/list", mediaServerHandler.OnlineList)
			serverAPI.GET("/media_server/one/:id", mediaServerHandler.One)
			serverAPI.GET("/media_server/check", mediaServerHandler.Check)
			serverAPI.GET("/media_server/record/check", mediaServerHandler.RecordCheck)
			serverAPI.POST("/media_server/save", mediaServerHandler.Save)
			serverAPI.DELETE("/media_server/delete", mediaServerHandler.Delete)
			serverAPI.GET("/media_server/media_info", mediaServerHandler.MediaInfo)
			serverAPI.GET("/media_server/load", mediaServerHandler.Load)
			serverAPI.GET("/system/configInfo", mediaServerHandler.SystemConfigInfo)
			serverAPI.GET("/gb_sip_config", gbSipConfigHandler.Get)
			serverAPI.POST("/gb_sip_config/save", gbSipConfigHandler.Save)
			serverAPI.GET("/object_store_config", objectStoreHandler.Get)
			serverAPI.POST("/object_store_config/save", objectStoreHandler.Save)
			serverAPI.GET("/object_store_config/health", objectStoreHandler.Health)
			serverAPI.GET("/resource/info", mediaServerHandler.ResourceInfo)
			serverAPI.GET("/info", mediaServerHandler.Info)
			serverAPI.GET("/map/config", mediaServerHandler.MapConfig)
			serverAPI.GET("/map/model-icon/list", mediaServerHandler.MapModelIconList)
		}

		onvif := auth.Group("/onvif")
		{
			onvif.POST("/device/discover", onvifHandler.Discover)
			onvif.GET("/device/discover", onvifHandler.Discover)
			onvif.GET("/device/query", onvifHandler.QueryDevices)
			onvif.POST("/device/add", onvifHandler.AddDevice)
			onvif.DELETE("/device/delete/:id", onvifHandler.DeleteDevice)
			onvif.POST("/device/sync/:id", onvifHandler.SyncDevice)
			onvif.POST("/device/probe", onvifHandler.Probe)
			onvif.GET("/channel/query", onvifHandler.QueryChannels)
			onvif.POST("/channel/update", onvifHandler.UpdateChannel)
			onvif.GET("/play/start", onvifHandler.StartPlay)
			onvif.GET("/play/stop", onvifHandler.StopPlay)
			onvif.POST("/ptz/control", onvifHandler.PTZControl)
			onvif.GET("/ptz/preset/query", onvifHandler.QueryPresets)
			onvif.GET("/ptz/preset/call", onvifHandler.GotoPreset)
			onvif.GET("/ptz/preset/add", onvifHandler.SetPreset)
			onvif.GET("/ptz/preset/delete", onvifHandler.RemovePreset)
		}

		// 统一设备接入门面（国标 + ONVIF）
		if deps.DeviceAccessService != nil {
			devices := auth.Group("/devices")
			{
				devices.GET("", deviceAccessHandler.List)
				devices.POST("", deviceAccessHandler.Create)
				devices.PUT("", deviceAccessHandler.Update)
				devices.DELETE("", deviceAccessHandler.Delete)
				devices.POST("/delete", deviceAccessHandler.Delete)
				devices.POST("/sync", deviceAccessHandler.Sync)
			}
		}

		jt1078 := auth.Group("/jt1078")
		{
			terminal := jt1078.Group("/terminal")
			{
				terminal.GET("/list", jt1078Handler.TerminalList)
				terminal.GET("/query", jt1078Handler.TerminalQuery)
				terminal.POST("/update", jt1078Handler.TerminalUpdate)
				terminal.POST("/add", jt1078Handler.TerminalAdd)
				terminal.DELETE("/delete", jt1078Handler.TerminalDelete)
				terminal.GET("/channel/list", jt1078Handler.ChannelList)
				terminal.POST("/channel/update", jt1078Handler.ChannelUpdate)
				terminal.POST("/channel/add", jt1078Handler.ChannelAdd)
			}
			jt1078.GET("/live/start", jt1078Handler.LiveStart)
			jt1078.GET("/live/stop", jt1078Handler.LiveStop)
			jt1078.GET("/talk/start", jt1078Handler.TalkStart)
			jt1078.GET("/talk/stop", jt1078Handler.TalkStop)
			jt1078.GET("/ptz", jt1078Handler.PTZ)
			jt1078.GET("/wiper", jt1078Handler.Wiper)
			jt1078.GET("/fill-light", jt1078Handler.FillLight)
			jt1078.GET("/record/list", jt1078Handler.RecordList)
			jt1078.GET("/playback/start", jt1078Handler.PlaybackStart)
			jt1078.GET("/playback/downloadUrl", jt1078Handler.PlaybackDownloadURL)
			jt1078.GET("/playback/control", jt1078Handler.PlaybackControl)
			jt1078.GET("/playback/stop", jt1078Handler.PlaybackStop)
			jt1078.GET("/playback/download", jt1078Handler.PlaybackDownload)
			jt1078.GET("/config/get", jt1078Handler.ConfigGet)
			jt1078.POST("/config/set", jt1078Handler.ConfigSet)
			jt1078.GET("/attribute", jt1078Handler.Attribute)
			jt1078.GET("/link-detection", jt1078Handler.LinkDetection)
			jt1078.GET("/position-info", jt1078Handler.PositionInfo)
			jt1078.POST("/text-msg", jt1078Handler.TextMsg)
			jt1078.GET("/telephone-callback", jt1078Handler.TelephoneCallback)
			jt1078.GET("/driver-information", jt1078Handler.DriverInformation)
			jt1078.POST("/control/factory-reset", jt1078Handler.FactoryReset)
			jt1078.POST("/control/reset", jt1078Handler.Reset)
			jt1078.POST("/control/connection", jt1078Handler.Connection)
			jt1078.GET("/control/door", jt1078Handler.ControlDoor)
			jt1078.GET("/media/attribute", jt1078Handler.MediaAttribute)
			jt1078.POST("/media/list", jt1078Handler.MediaList)
			jt1078.POST("/set-phone-book", jt1078Handler.SetPhoneBook)
			jt1078.POST("/shooting", jt1078Handler.Shooting)
			jt1078.GET("/snap", jt1078Handler.Snap)
			jt1078.GET("/media/upload/one/upload", jt1078Handler.MediaUpload)
		}

		deviceQuery := auth.Group("/device/query")
		{
			deviceQuery.GET("/devices/:deviceId", deviceHandler.GetDevice)
			deviceQuery.GET("/devices", deviceHandler.ListDevices)
			deviceQuery.GET("/devices/:deviceId/channels", deviceHandler.ListChannels)
			deviceQuery.GET("/devices/:deviceId/sync", deviceHandler.SyncDevice)
			deviceQuery.DELETE("/devices/:deviceId/delete", deviceHandler.DeleteDevice)
			deviceQuery.GET("/sync_status", deviceHandler.SyncStatus)
			deviceQuery.POST("/device/add", deviceHandler.AddDevice)
			deviceQuery.POST("/device/update", deviceHandler.UpdateDevice)
			deviceQuery.GET("/channel/one", deviceHandler.GetChannel)
			deviceQuery.POST("/transport/:deviceId/:streamMode", deviceHandler.SetTransport)
			deviceQuery.GET("/subscribe/catalog", deviceHandler.SubscribeCatalog)
			deviceQuery.GET("/subscribe/mobile-position", deviceHandler.SubscribeMobilePosition)
			deviceQuery.GET("/subscribe/alarm", deviceHandler.SubscribeAlarm)
			deviceQuery.GET("/statistics/keepalive", deviceHandler.KeepaliveStatistics)
			deviceQuery.GET("/statistics/register", deviceHandler.RegisterStatistics)
			deviceQuery.POST("/channel/audio", deviceHandler.ChangeChannelAudio)
			deviceQuery.GET("/snap/:deviceId/:channelId", snapHandler.GetChannelSnap)
			deviceQuery.POST("/snap/:deviceId/:channelId", snapHandler.UploadChannelSnap)
		}

		auth.GET("/play/start/:deviceId/:channelId", playHandler.Start)
		auth.GET("/play/stop/:deviceId/:channelId", playHandler.Stop)
		auth.GET("/play/broadcast/:deviceId/:channelId", playHandler.Broadcast)
		auth.GET("/play/broadcast/stop/:deviceId/:channelId", playHandler.BroadcastStop)

		auth.GET("/playback/start/:deviceId/:channelId", playbackHandler.Start)
		auth.GET("/playback/stop/:deviceId/:channelId/:streamId", playbackHandler.Stop)
		auth.GET("/playback/pause/:streamId", playbackHandler.Pause)
		auth.GET("/playback/resume/:streamId", playbackHandler.Resume)
		auth.GET("/playback/speed/:streamId/:speed", playbackHandler.Speed)
		auth.GET("/playback/seek/:streamId/:seekTime", playbackHandler.Seek)

		gbRecord := auth.Group("/gb_record")
		{
			gbRecord.GET("/query/:deviceId/:channelId", gbRecordHandler.Query)
			gbRecord.GET("/download/start/:deviceId/:channelId", gbRecordHandler.DownloadStart)
			gbRecord.GET("/download/stop/:deviceId/:channelId/:stream", gbRecordHandler.DownloadStop)
			gbRecord.GET("/download/progress/:deviceId/:channelId/:stream", gbRecordHandler.DownloadProgress)
		}

		cloudRecord := auth.Group("/cloud/record")
		{
			cloudRecord.GET("/list", cloudRecordHandler.List)
			cloudRecord.GET("/date/list", cloudRecordHandler.DateList)
			cloudRecord.DELETE("/delete", cloudRecordHandler.Delete)
			cloudRecord.GET("/play/path", cloudRecordHandler.PlayPath)
			cloudRecord.GET("/loadRecord", cloudRecordHandler.LoadRecord)
			cloudRecord.GET("/seek", cloudRecordHandler.Seek)
			cloudRecord.GET("/speed", cloudRecordHandler.Speed)
			cloudRecord.GET("/task/add", cloudRecordHandler.AddTask)
			cloudRecord.GET("/task/list", cloudRecordHandler.TaskList)
		}

		logAPI := auth.Group("/log")
		{
			logAPI.GET("/list", logHandler.List)
			logAPI.GET("/file/:fileName", logHandler.File)
		}

		push := auth.Group("/push")
		{
			push.GET("/list", streamPushHandler.List)
			push.POST("/add", streamPushHandler.Add)
			push.POST("/update", streamPushHandler.Update)
			push.POST("/remove", streamPushHandler.Remove)
			push.DELETE("/batchRemove", streamPushHandler.BatchRemove)
			push.POST("/save_to_gb", streamPushHandler.SaveToGB)
			push.DELETE("/remove_form_gb", streamPushHandler.RemoveFromGB)
			push.GET("/start", streamPushHandler.Start)
		}

		proxy := auth.Group("/proxy")
		{
			proxy.GET("/list", streamProxyHandler.List)
			proxy.POST("/add", streamProxyHandler.Add)
			proxy.POST("/save", streamProxyHandler.Save)
			proxy.POST("/update", streamProxyHandler.Update)
			proxy.DELETE("/delete", streamProxyHandler.Delete)
			proxy.GET("/start", streamProxyHandler.Start)
			proxy.GET("/stop", streamProxyHandler.Stop)
			proxy.GET("/ffmpeg_cmd/list", streamProxyHandler.FFmpegCmdList)
		}

		recordPlan := auth.Group("/record/plan")
		{
			recordPlan.POST("/add", recordPlanHandler.Add)
			recordPlan.POST("/update", recordPlanHandler.Update)
			recordPlan.GET("/get", recordPlanHandler.Get)
			recordPlan.GET("/query", recordPlanHandler.Query)
			recordPlan.DELETE("/delete", recordPlanHandler.Delete)
			recordPlan.POST("/link", recordPlanHandler.Link)
			recordPlan.GET("/channel/list", recordPlanHandler.ChannelList)
		}

		alarm := auth.Group("/alarm")
		{
			alarm.GET("/list", alarmHandler.List)
			alarm.DELETE("/delete", alarmHandler.Delete)
			alarm.DELETE("/clear", alarmHandler.Clear)
		}
		auth.GET("/alarm/snap/:id", alarmHandler.Snap)

		platform := auth.Group("/platform")
		{
			platform.GET("/server_config", platformHandler.ServerConfig)
			platform.GET("/query", platformHandler.Query)
			platform.POST("/add", platformHandler.Add)
			platform.POST("/update", platformHandler.Update)
			platform.DELETE("/delete", platformHandler.Delete)
			platform.GET("/exit/:deviceGbId", platformHandler.Exit)
			platform.GET("/channel/push", platformHandler.PushChannel)
			platform.GET("/channel/list", platformHandler.ChannelList)
			platform.POST("/channel/add", platformHandler.ChannelAdd)
			platform.POST("/channel/device/add", platformHandler.ChannelDeviceAdd)
			platform.POST("/channel/device/remove", platformHandler.ChannelDeviceRemove)
			platform.DELETE("/channel/remove", platformHandler.ChannelRemove)
			platform.POST("/channel/custom/update", platformHandler.ChannelCustomUpdate)
		}

		subordinate := auth.Group("/subordinate")
		{
			subordinate.GET("/query", subordinateHandler.List)
			subordinate.GET("/one/:id", subordinateHandler.Get)
			subordinate.POST("/add", subordinateHandler.Add)
			subordinate.POST("/update/:id", subordinateHandler.Update)
			subordinate.DELETE("/delete/:id", subordinateHandler.Delete)
		}

		position := auth.Group("/position")
		{
			position.GET("/history/:deviceId", positionHandler.History)
			position.GET("/latest", positionHandler.Latest)
		}

		frontEnd := auth.Group("/front-end")
		{
			frontEnd.GET("/ptz/:deviceId/:channelId", ptzHandler.PTZ)
			frontEnd.GET("/preset/query/:deviceId/:channelId", ptzHandler.QueryPreset)
			frontEnd.GET("/preset/add/:deviceId/:channelId", ptzHandler.AddPreset)
			frontEnd.GET("/preset/call/:deviceId/:channelId", ptzHandler.CallPreset)
			frontEnd.GET("/preset/delete/:deviceId/:channelId", ptzHandler.DeletePreset)
		}

		common := auth.Group("/common/channel")
		{
			common.GET("/one", commonChannelHandler.One)
			common.GET("/list", commonChannelHandler.List)
			common.POST("/add", commonChannelHandler.Add)
			common.POST("/update", commonChannelHandler.Update)
			common.POST("/reset", commonChannelHandler.Reset)
			common.GET("/industry/list", commonChannelHandler.IndustryList)
			common.GET("/type/list", commonChannelHandler.TypeList)
			common.GET("/network/identification/list", commonChannelHandler.NetworkList)
			common.GET("/play", commonChannelHandler.Play)
			common.GET("/play/stop", commonChannelHandler.PlayStop)
			common.GET("/talk/start", commonChannelHandler.TalkStart)
			common.GET("/talk/stop", commonChannelHandler.TalkStop)
			common.GET("/broadcast/start", commonChannelHandler.BroadcastStart)
			common.GET("/broadcast/stop", commonChannelHandler.BroadcastStop)
			common.GET("/front-end/ptz", commonChannelHandler.PTZ)
			common.GET("/front-end/preset/query", commonChannelHandler.QueryPreset)
			common.GET("/front-end/preset/add", commonChannelHandler.AddPreset)
			common.GET("/front-end/preset/call", commonChannelHandler.CallPreset)
			common.GET("/front-end/preset/delete", commonChannelHandler.DeletePreset)
			common.GET("/playback/query", commonChannelHandler.PlaybackQuery)
			common.GET("/playback", commonChannelHandler.Playback)
			common.GET("/playback/stop", commonChannelHandler.PlaybackStop)
			common.GET("/playback/pause", commonChannelHandler.PlaybackPause)
			common.GET("/playback/resume", commonChannelHandler.PlaybackResume)
			common.GET("/playback/speed", commonChannelHandler.PlaybackSpeed)
			common.GET("/playback/seek", commonChannelHandler.PlaybackSeek)
			common.GET("/map/list", commonChannelHandler.MapList)
			common.GET("/map/tile/:z/:x/:y", commonChannelHandler.MapTile)
			common.GET("/map/thin/tile/:z/:x/:y", commonChannelHandler.MapThinTile)
			common.POST("/map/reset-level", commonChannelHandler.MapResetLevel)
			common.POST("/map/thin/draw", commonChannelHandler.MapThinDraw)
			common.GET("/map/thin/clear", commonChannelHandler.MapThinClear)
			common.GET("/map/thin/save", commonChannelHandler.MapThinSave)
			common.GET("/map/thin/progress", commonChannelHandler.MapThinProgress)
			common.POST("/region/add", commonChannelHandler.RegionAdd)
			common.POST("/region/delete", commonChannelHandler.RegionDelete)
			common.POST("/region/device/add", commonChannelHandler.RegionDeviceAdd)
			common.POST("/region/device/delete", commonChannelHandler.RegionDeviceDelete)
			common.POST("/group/add", commonChannelHandler.GroupAdd)
			common.POST("/group/delete", commonChannelHandler.GroupDelete)
			common.POST("/group/device/add", commonChannelHandler.GroupDeviceAdd)
			common.POST("/group/device/delete", commonChannelHandler.GroupDeviceDelete)
			common.GET("/civilcode/list", commonChannelHandler.CivilCodeList)
			common.GET("/parent/list", commonChannelHandler.ParentList)
		}

		group := auth.Group("/group")
		{
			group.GET("/tree/list", groupHandler.TreeList)
			group.GET("/tree/query", groupHandler.TreeQuery)
			group.POST("/add", groupHandler.Add)
			group.POST("/update", groupHandler.Update)
			group.DELETE("/delete", groupHandler.Delete)
			group.GET("/path", groupHandler.Path)
		}

		region := auth.Group("/region")
		{
			region.GET("/tree/list", regionHandler.TreeList)
			region.GET("/tree/query", regionHandler.TreeQuery)
			region.POST("/add", regionHandler.Add)
			region.POST("/update", regionHandler.Update)
			region.DELETE("/delete", regionHandler.Delete)
			region.GET("/path", regionHandler.Path)
			region.GET("/description", regionHandler.Description)
			region.GET("/addByCivilCode", regionHandler.AddByCivilCode)
			region.GET("/base/child/list", regionHandler.BaseChildList)
		}
	}
}
