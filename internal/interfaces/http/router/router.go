package router

import (
	alarmapp "zero-web-server/internal/application/alarm"
	assetapp "zero-web-server/internal/application/asset"
	appauth "zero-web-server/internal/application/auth"
	channelapp "zero-web-server/internal/application/channel"
	cloudrecordapp "zero-web-server/internal/application/cloudrecord"
	deviceapp "zero-web-server/internal/application/device"
	deviceaccess "zero-web-server/internal/application/deviceaccess"
	downstreamapp "zero-web-server/internal/application/downstream"
	gbsipconfig "zero-web-server/internal/application/gbsipconfig"
	groupapp "zero-web-server/internal/application/group"
	ipcdevice "zero-web-server/internal/application/ipcdevice"
	mediaserverapp "zero-web-server/internal/application/mediaserver"
	objectstoreapp "zero-web-server/internal/application/objectstore"
	onvifapp "zero-web-server/internal/application/onvif"
	"zero-web-server/internal/application/ops"
	playapp "zero-web-server/internal/application/play"
	playbackapp "zero-web-server/internal/application/playback"
	pluginapp "zero-web-server/internal/application/plugin"
	positionapp "zero-web-server/internal/application/position"
	ptzapp "zero-web-server/internal/application/ptz"
	publishauth "zero-web-server/internal/application/publishauth"
	recordplanapp "zero-web-server/internal/application/recordplan"
	regionapp "zero-web-server/internal/application/region"
	snapapp "zero-web-server/internal/application/snap"
	streamproxyapp "zero-web-server/internal/application/streamproxy"
	streampushapp "zero-web-server/internal/application/streampush"
	upstreamapp "zero-web-server/internal/application/upstream"
	"zero-web-server/internal/infrastructure/config"
	"zero-web-server/internal/infrastructure/persistence"
	"zero-web-server/internal/interfaces/hook"
	"zero-web-server/internal/interfaces/http/handler"
	"zero-web-server/internal/interfaces/http/middleware"
	jwtmgr "zero-web-server/pkg/jwt"

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
	PluginService       *pluginapp.Service
	IPCService          *ipcdevice.Service
	PluginToken         string
	AssetService        *assetapp.Service
}

func Setup(r *gin.Engine, deps Deps) {
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	// WebRTC 信令反代：始终挂载，上游基址按请求动态解析（媒体节点可能在平台启动后才自注册）。
	resolveMediaBase := func() string {
		if deps.MediaServerService != nil {
			if u := deps.MediaServerService.FirstOnlineBaseURL(); u != "" {
				return u
			}
		}
		return deps.MediaBaseURL
	}
	webrtcProxy := handler.NewWebRTCProxyHandler(resolveMediaBase)
	r.Any("/index/api/webrtc", webrtcProxy.Proxy)
	r.Any("/index/api/whep", webrtcProxy.Proxy)
	r.Any("/index/api/whep/*path", webrtcProxy.Proxy)
	r.Any("/index/api/whip", webrtcProxy.Proxy)

	userHandler := handler.NewUserHandler(deps.AuthService)
	healthHandler := handler.NewHealthHandler(deps.Version)
	serverHandler := handler.NewServerHandler(deps.ServerID, deps.Metrics)
	onvifHandler := handler.NewONVIFHandler(deps.ONVIFService, deps.AssetService)
	deviceHandler := handler.NewDeviceHandler(deps.DeviceService)
	deviceAccessHandler := handler.NewDeviceAccessHandler(deps.DeviceAccessService, deps.AssetService)
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
	pluginHandler := handler.NewPluginHandler(deps.PluginService, deps.IPCService, deps.AssetService, deps.PluginToken)
	assetHandler := handler.NewAssetHandler(deps.AssetService)
	mediaHook := hook.NewHandler(
		deps.PlayService, deps.PlaybackService, deps.CloudRecordService,
		deps.StreamPushService, deps.StreamProxyService, deps.RecordPlanService,
		deps.MediaServerService, deps.PublishAuth, deps.StreamOnDemand,
	)

	r.GET("/health", healthHandler.Health)
	// 实时日志 WebSocket（/channel/log；token 走子协议）
	r.GET("/channel/log", handler.LogChannel)
	api := r.Group("/api/v1")
	{
		registerUser(api, userHandler)
		registerServer(api, healthHandler, serverHandler)
		registerPluginPublic(api, pluginHandler)
	}

	hookGroup := r.Group("/index/hook")
	mediaHook.Register(hookGroup)

	auth := r.Group("/api/v1")
	auth.Use(middleware.Auth(deps.JWT, deps.UserRepo))
	{
		auth.POST("/user/userInfo", userHandler.UserInfo)
		auth.POST("/user/changePassword", userHandler.ChangePassword)

		registerPlugin(auth, pluginHandler)
		registerAssets(auth, assetHandler)

		registerUserMgmt(auth, userHandler, roleHandler)

		registerServerAPI(auth, mediaServerHandler, gbSipConfigHandler, objectStoreHandler)

		registerOnvif(auth, onvifHandler)

		// 统一设备接入门面（国标 + ONVIF）
		if deps.DeviceAccessService != nil {
			registerDevices(auth, deviceAccessHandler)
		}

		registerJt1078(auth, jt1078Handler)

		registerDeviceQuery(auth, deviceHandler, snapHandler)

		registerDeviceControl(auth, deviceHandler)

		registerDeviceConfig(auth, deviceHandler)

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

		registerGbRecord(auth, gbRecordHandler)

		registerCloudRecord(auth, cloudRecordHandler)

		registerLogAPI(auth, logHandler)

		registerPush(auth, streamPushHandler)

		registerProxy(auth, streamProxyHandler)

		registerRecordPlan(auth, recordPlanHandler)

		registerAlarm(auth, alarmHandler)
		auth.GET("/alarm/snap/:id", alarmHandler.Snap)

		registerPlatform(auth, platformHandler)

		registerSubordinate(auth, subordinateHandler)

		registerPosition(auth, positionHandler)

		registerFrontEnd(auth, ptzHandler)

		registerCommon(auth, commonChannelHandler)

		registerGroup(auth, groupHandler)

		registerRegion(auth, regionHandler)
	}
}
