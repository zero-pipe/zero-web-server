package router

import (
	accesstokenapp "zero-web-server/internal/application/accesstoken"
	agentapp "zero-web-server/internal/application/agent"
	alarmapp "zero-web-server/internal/application/alarm"
	assetapp "zero-web-server/internal/application/asset"
	appauth "zero-web-server/internal/application/auth"
	channelapp "zero-web-server/internal/application/channel"
	cloudrecordapp "zero-web-server/internal/application/cloudrecord"
	customeventapp "zero-web-server/internal/application/customevent"
	deviceapp "zero-web-server/internal/application/device"
	deviceaccess "zero-web-server/internal/application/deviceaccess"
	dictapp "zero-web-server/internal/application/dict"
	downstreamapp "zero-web-server/internal/application/downstream"
	eventingest "zero-web-server/internal/application/eventingest"
	eventsubapp "zero-web-server/internal/application/eventsub"
	gbsipconfig "zero-web-server/internal/application/gbsipconfig"
	groupapp "zero-web-server/internal/application/group"
	infraconfigapp "zero-web-server/internal/application/infraconfig"
	ipcdevice "zero-web-server/internal/application/ipcdevice"
	mediaserverapp "zero-web-server/internal/application/mediaserver"
	mediasessionapp "zero-web-server/internal/application/mediasession"
	objectstoreapp "zero-web-server/internal/application/objectstore"
	onvifapp "zero-web-server/internal/application/onvif"
	"zero-web-server/internal/application/ops"
	playapp "zero-web-server/internal/application/play"
	playbackapp "zero-web-server/internal/application/playback"
	pluginapp "zero-web-server/internal/application/plugin"
	pluginpkgapp "zero-web-server/internal/application/pluginpkgapp"
	positionapp "zero-web-server/internal/application/position"
	publishauth "zero-web-server/internal/application/publishauth"
	recordplanapp "zero-web-server/internal/application/recordplan"
	regionapp "zero-web-server/internal/application/region"
	snapapp "zero-web-server/internal/application/snap"
	storagenodeapp "zero-web-server/internal/application/storagenode"
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
	AuthService          *appauth.Service
	ONVIFService         *onvifapp.Service
	DeviceService        *deviceapp.Service
	DeviceAccessService  *deviceaccess.Service
	PlayService          *playapp.Service
	PlaybackService      *playbackapp.Service
	AlarmService         *alarmapp.Service
	PlatformService      *upstreamapp.Service
	PlatformChannelSvc   *upstreamapp.ChannelService
	SubordinateService   *downstreamapp.Service
	PositionService      *positionapp.Service
	CloudRecordService   *cloudrecordapp.Service
	StreamPushService    *streampushapp.Service
	StreamProxyService   *streamproxyapp.Service
	MediaSessionService  *mediasessionapp.Service
	RecordPlanService    *recordplanapp.Service
	MediaServerService   *mediaserverapp.Service
	StorageNodeService   *storagenodeapp.Service
	CommonChannelSvc     *channelapp.Service
	GroupService         *groupapp.Service
	RegionService        *regionapp.Service
	UserRepo             *persistence.UserRepository
	PublishAuth          *publishauth.PublishAuth
	StreamOnDemand       bool
	MediaBaseURL         string
	JWT                  *jwtmgr.Manager
	ServerID             string
	Version              string
	PlayTimeoutMs        int
	RecordInfoTimeoutMs  int
	SIPConfig            config.SIPConfig
	GbSipConfigService   *gbsipconfig.Service
	ObjectStoreService   *objectstoreapp.Service
	SnapService          *snapapp.Service
	ServerPort           int
	MediaIP              string
	LogDir               string
	Metrics              *ops.Metrics
	Dashboard            *ops.Dashboard
	PluginService        *pluginapp.Service
	IPCService           *ipcdevice.Service
	PluginToken          string
	AgentService         *agentapp.Service
	PluginPackageService *pluginpkgapp.Service
	AgentToken           string
	AssetService         *assetapp.Service
	InfraService         *infraconfigapp.Service
	EventSubService      *eventsubapp.Service
	EventService         *eventingest.Service
	CustomEventService   *customeventapp.Service
	AccessTokenService   *accesstokenapp.Service
	DictService          *dictapp.Service
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
	deviceHandler := handler.NewDeviceHandler(deps.DeviceService).WithAssetPruner(deps.AssetService)
	deviceAccessHandler := handler.NewDeviceAccessHandler(deps.DeviceAccessService, deps.AssetService)
	gbRecordHandler := handler.NewGBRecordHandler(deps.PlaybackService, deps.RecordInfoTimeoutMs)
	alarmHandler := handler.NewAlarmHandler(deps.AlarmService, deps.ObjectStoreService)
	platformHandler := handler.NewPlatformHandler(deps.PlatformService, deps.PlatformChannelSvc)
	subordinateHandler := handler.NewSubordinateHandler(deps.SubordinateService)
	positionHandler := handler.NewPositionHandler(deps.PositionService)
	cloudRecordHandler := handler.NewCloudRecordHandler(deps.CloudRecordService, deps.PlayTimeoutMs)
	streamPushHandler := handler.NewStreamPushHandler(deps.StreamPushService, deps.PlayTimeoutMs)
	streamProxyHandler := handler.NewStreamProxyHandler(deps.StreamProxyService, deps.PlayTimeoutMs)
	mediaSessionHandler := handler.NewMediaSessionHandler(deps.MediaSessionService)
	recordPlanHandler := handler.NewRecordPlanHandler(deps.RecordPlanService)
	storageNodeHandler := handler.NewStorageNodeHandler(deps.StorageNodeService)
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
	commonChannelHandler := handler.NewCommonChannelHandler(deps.CommonChannelSvc)
	groupHandler := handler.NewGroupHandler(deps.GroupService)
	regionHandler := handler.NewRegionHandler(deps.RegionService)
	roleHandler := handler.NewRoleHandler(deps.AuthService)
	logHandler := handler.NewLogHandler(ops.NewLogService(deps.LogDir))
	pluginHandler := handler.NewPluginHandler(deps.PluginService, deps.IPCService, deps.AssetService)
	pluginPackageHandler := handler.NewPluginPackageHandler(deps.PluginPackageService, deps.AgentToken)
	agentHandler := handler.NewAgentHandler(deps.AgentService, deps.AgentToken)
	eventSubHandler := handler.NewEventSubHandler(deps.EventSubService, deps.InfraService)
	eventHandler := handler.NewEventHandler(deps.EventService)
	customEventHandler := handler.NewCustomEventHandler(deps.CustomEventService)
	assetHandler := handler.NewAssetHandler(deps.AssetService)
	infraHandler := handler.NewInfraHandler(deps.InfraService)
	accessTokenHandler := handler.NewAccessTokenHandler(deps.AccessTokenService)
	dictHandler := handler.NewDictHandler(deps.DictService)
	var accessTokenCheck func(token string) (bool, bool)
	if deps.AccessTokenService != nil {
		accessTokenCheck = func(token string) (bool, bool) {
			enforcing := len(deps.AccessTokenService.EnabledTokens()) > 0
			return deps.AccessTokenService.Verify(token), enforcing
		}
	}
	mediaHook := hook.NewHandler(
		deps.PlayService, deps.PlaybackService, deps.CloudRecordService,
		deps.StreamPushService, deps.StreamProxyService, deps.RecordPlanService,
		deps.MediaServerService, deps.StorageNodeService, deps.PublishAuth, deps.StreamOnDemand,
		accessTokenCheck,
	)

	r.GET("/health", healthHandler.Health)
	// 实时日志 WebSocket（/channel/log；token 走子协议）
	r.GET("/channel/log", handler.LogChannel)
	api := r.Group("/api/v1")
	{
		registerUser(api, userHandler)
		registerServer(api, healthHandler, serverHandler)
		registerAgentPublic(api, agentHandler)
		registerPluginPackagePublic(api, pluginPackageHandler)
		// GB28181 图像抓拍：设备按 UploadURL 公开上传（2022 9.14）
		api.POST("/gb/snapshot/upload", snapHandler.UploadGBSnapshot)
	}

	hookGroup := r.Group("/index/hook")
	mediaHook.Register(hookGroup)

	auth := r.Group("/api/v1")
	auth.Use(middleware.Auth(deps.JWT, deps.UserRepo))
	{
		auth.POST("/user/userInfo", userHandler.UserInfo)
		auth.POST("/user/changePassword", userHandler.ChangePassword)

		registerPlugin(auth, pluginHandler)
		registerPluginPackages(auth, pluginPackageHandler)
		registerAgents(auth, agentHandler)
		registerEventSub(auth, eventSubHandler)
		registerEvents(auth, eventHandler)
		registerCustomEvents(auth, customEventHandler)
		registerAssets(auth, assetHandler)
		registerInfra(auth, infraHandler)

		registerUserMgmt(auth, userHandler, roleHandler)

		registerServerAPI(auth, mediaServerHandler, gbSipConfigHandler, objectStoreHandler)

		registerAccessToken(auth, accessTokenHandler)

		registerDict(auth, dictHandler)

		// 设备接入唯一命名空间：/devices（统一）+ /devices/{gb,onvif,plugin}（协议专属动作）
		registerDevices(auth, deviceAccessHandler, deviceHandler, onvifHandler, pluginHandler, snapHandler)

		// 直播/回放/云台统一收敛到 /resources/{id}/*（见 registerAssets / registerPlaybackCloud）。
		registerGbRecord(auth, gbRecordHandler)

		registerCloudRecord(auth, cloudRecordHandler)
		registerPlaybackCloud(auth, cloudRecordHandler)

		registerLogAPI(auth, logHandler)

		registerPush(auth, streamPushHandler)

		registerProxy(auth, streamProxyHandler)

		// 国标收流列表（dashboard 弹窗）：返回在线 GB 通道，播放仍走 resources 族。
		auth.GET("/devices/gb/query/streams", mediaSessionHandler.GBReceivingChannels)
		registerMediaSessions(auth, mediaSessionHandler)

		registerRecordPlan(auth, recordPlanHandler)
		auth.GET("/storage/node/list", storageNodeHandler.List)

		registerAlarm(auth, alarmHandler)
		auth.GET("/alarm/snap/:id", alarmHandler.Snap)

		registerPlatform(auth, platformHandler)

		registerSubordinate(auth, subordinateHandler)

		registerPosition(auth, positionHandler)

		registerCommon(auth, commonChannelHandler)

		registerGroup(auth, groupHandler)

		registerRegion(auth, regionHandler)
	}
}
