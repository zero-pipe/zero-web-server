package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	ipcplugin "zero-web-server/internal/adapter/ipcplugin"
	mediacluster "zero-web-server/internal/adapter/mediacluster"
	"zero-web-server/internal/adapter/mqttbus"
	accesstokenapp "zero-web-server/internal/application/accesstoken"
	agentapp "zero-web-server/internal/application/agent"
	alarmapp "zero-web-server/internal/application/alarm"
	assetapp "zero-web-server/internal/application/asset"
	appauth "zero-web-server/internal/application/auth"
	cascadeapp "zero-web-server/internal/application/cascade"
	channelapp "zero-web-server/internal/application/channel"
	"zero-web-server/internal/application/channeldrv"
	cloudrecordapp "zero-web-server/internal/application/cloudrecord"
	customeventapp "zero-web-server/internal/application/customevent"
	deviceapp "zero-web-server/internal/application/device"
	deviceaccess "zero-web-server/internal/application/deviceaccess"
	dictapp "zero-web-server/internal/application/dict"
	downstreamapp "zero-web-server/internal/application/downstream"
	eventingestapp "zero-web-server/internal/application/eventingest"
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
	ptzapp "zero-web-server/internal/application/ptz"
	publishauth "zero-web-server/internal/application/publishauth"
	recordplanapp "zero-web-server/internal/application/recordplan"
	regionapp "zero-web-server/internal/application/region"
	snapapp "zero-web-server/internal/application/snap"
	storagenodeapp "zero-web-server/internal/application/storagenode"
	streamproxyapp "zero-web-server/internal/application/streamproxy"
	streampushapp "zero-web-server/internal/application/streampush"
	upstreamapp "zero-web-server/internal/application/upstream"
	"zero-web-server/internal/civilcode"
	"zero-web-server/internal/infrastructure/config"
	onvifinfra "zero-web-server/internal/infrastructure/onvif"
	"zero-web-server/internal/infrastructure/persistence"
	"zero-web-server/internal/infrastructure/persistence/mysql"
	redisinfra "zero-web-server/internal/infrastructure/redis"
	sipinfra "zero-web-server/internal/infrastructure/sip"
	"zero-web-server/internal/interfaces/http/router"
	"zero-web-server/internal/mediakit"
	"zero-web-server/internal/port"
	jwtmgr "zero-web-server/pkg/jwt"
	applog "zero-web-server/pkg/log"

	"github.com/gin-gonic/gin"
)

const version = "1.0.0"

func main() {
	configPath := flag.String("config", "configs/config.yaml", "config file path")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		applog.Fatalf("load config: %v", err)
	}
	// 行政区划 CSV 默认与 config.yaml 同目录（运行时读取，不 embed）
	civilcode.Configure(filepath.Join(filepath.Dir(*configPath), "civilCode.csv"))
	if err := applog.Init(cfg.Log); err != nil {
		applog.Fatalf("init log: %v", err)
	}
	if err := civilcode.LoadError(); err != nil {
		applog.Warn("civilcode csv not loaded", "err", err)
	}
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := mysql.NewDB(cfg.MySQL)
	if err != nil {
		applog.Fatalf("init mysql: %v", err)
	}

	var redisClient *redisinfra.Client
	redisClient, err = redisinfra.NewClient(cfg.Redis)
	if err != nil {
		applog.Warn("redis unavailable", "err", err)
	} else {
		defer redisClient.Close()
	}

	jwkPath := cfg.UserSettings.JWKFile
	if !filepath.IsAbs(jwkPath) {
		jwkPath = filepath.Join(".", jwkPath)
	}
	jwtManager, err := jwtmgr.NewManager(jwkPath, cfg.UserSettings.LoginTimeout)
	if err != nil {
		applog.Fatalf("init jwt: %v", err)
	}

	userRepo := persistence.NewUserRepository(db)
	deviceRepo := persistence.NewDeviceRepository(db)
	channelRepo := persistence.NewChannelRepository(db)
	onvifDeviceRepo := persistence.NewOnvifDeviceRepository(db)
	onvifChannelRepo := persistence.NewOnvifChannelRepository(db)
	alarmRepo := persistence.NewAlarmRepository(db)
	positionRepo := persistence.NewPositionRepository(db)
	platformRepo := persistence.NewPlatformRepository(db)
	cloudRecordRepo := persistence.NewCloudRecordRepository(db)
	streamPushRepo := persistence.NewStreamPushRepository(db)
	streamProxyRepo := persistence.NewStreamProxyRepository(db)
	recordPlanRepo := persistence.NewRecordPlanRepository(db)
	mediaServerRepo := persistence.NewMediaServerRepository(db)
	storageNodeRepo := persistence.NewStorageNodeRepository(db)
	groupRegionRepo := persistence.NewGroupRegionRepository(db)
	gbSipConfigRepo := persistence.NewGbSipConfigRepository(db)
	gbSipConfigService := gbsipconfig.NewService(gbSipConfigRepo, cfg.SIP)

	// 库空则用 yaml 默认 SIP 写入库；之后以库为准，页面改编码可热更新
	sipCfg, err := gbSipConfigService.Bootstrap()
	if err != nil {
		applog.Fatalf("load gb sip config: %v", err)
	}

	authService := appauth.NewService(userRepo, jwtManager, cfg.UserSettings.ServerID)
	var deviceCache port.DeviceCache = port.NoopDeviceCache{}
	if redisClient != nil {
		deviceCache = redisClient
	}
	deviceService := deviceapp.NewService(deviceRepo, channelRepo, deviceCache)
	requirePreRegister := cfg.GB.RequirePreRegister
	if row, err := gbSipConfigService.GetOrEmpty(); err == nil && row != nil {
		// 库内开关优先（页面可改）；种子行默认 true
		requirePreRegister = row.RequirePreRegister
	}
	deviceService.SetRequirePreRegister(requirePreRegister)
	publishRegistry := publishauth.NewPublishRegistry()
	publishAuth := publishauth.NewPublishAuth(
		streamPushRepo, streamProxyRepo,
		cfg.UserSettings.RecordPushLive,
		publishRegistry,
	)
	// 媒体节点以数据库为准；启动时允许为空，由页面动态添加
	mediaServerService := mediaserverapp.NewService(mediaServerRepo, cfg.UserSettings.ServerID)
	storageNodeService := storagenodeapp.NewService(storageNodeRepo, cfg.UserSettings.ServerID)
	mediaCluster := mediacluster.New(mediaServerService)

	// 对外接入密钥：可视化配置 + 同步到媒体节点本地验签。
	accessTokenRepo := persistence.NewAccessTokenRepository(db)
	accessTokenService := accesstokenapp.NewService(accessTokenRepo)
	dictRepo := persistence.NewDictRepository(db)
	dictService := dictapp.NewService(dictRepo)
	// token 只在启用集合变化时推送：节点每次心跳都会触发重推，若每 30s 都推一遍空列表，
	// ZMS 会持续打 "token list cleared" 噪声日志。
	var (
		tokenMu   sync.Mutex
		tokenSeen = map[string]string{}
	)
	syncTokensToNode := func(nodeID string) {
		if nodeID == "" {
			return
		}
		node, err := mediaServerService.Lookup(nodeID)
		if err != nil || node == nil || node.Client == nil {
			return
		}
		tokens := accessTokenService.EnabledTokens()
		sig := strings.Join(tokens, ",")
		tokenMu.Lock()
		if tokenSeen[nodeID] == sig {
			tokenMu.Unlock()
			return
		}
		tokenSeen[nodeID] = sig
		tokenMu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := node.Client.SetAuthTokens(ctx, tokens); err != nil {
			// 推送失败则清掉记录，下次心跳重试。
			tokenMu.Lock()
			delete(tokenSeen, nodeID)
			tokenMu.Unlock()
			applog.Warnf("[access-token] 同步 token 到媒体节点 %s 失败: %v", nodeID, err)
		}
	}
	syncTokensToAll := func(tokens []string) {
		nodes, err := mediaServerService.ListOnline()
		if err != nil {
			return
		}
		for _, v := range nodes {
			go func(ip string, port int, secret string) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if _, err := mediakit.NewClientAddr(ip, port, secret).SetAuthTokens(ctx, tokens); err != nil {
					applog.Warnf("[access-token] 同步 token 到节点 %s:%d 失败: %v", ip, port, err)
				}
			}(v.IP, v.HTTPPort, v.Secret)
		}
	}
	accessTokenService.SetOnChange(syncTokensToAll)
	mediaServerService.SetOnRegistered(syncTokensToNode)

	pluginRepo := persistence.NewPluginRepository(db)
	pluginService := pluginapp.NewService(pluginRepo)
	pluginPackageRepo := persistence.NewPluginPackageRepository(db)
	pluginToken := os.Getenv("ZWS_PLUGIN_TOKEN")
	agentToken := os.Getenv("ZWS_AGENT_TOKEN")
	agentRepo := persistence.NewAgentRepository(db)
	agentService := agentapp.NewService(agentRepo, pluginRepo, agentToken)
	ipcService := ipcdevice.NewService(pluginService, pluginRepo, mediaCluster, ipcplugin.NewFactory(pluginToken), cfg.UserSettings.ServerID)
	agentService.SetDeviceCounter(ipcService)

	assetRepo := persistence.NewAssetRepository(db)
	assetProjector := persistence.NewAssetProjector(db)
	if err := assetProjector.Backfill(); err != nil {
		applog.Warn("asset backfill", "err", err)
	}

	objectStoreRepo := persistence.NewObjectStoreConfigRepository(db)
	objectStoreService := objectstoreapp.NewService(objectStoreRepo)
	snapService := snapapp.NewService(objectStoreService)
	pluginPackageService := pluginpkgapp.NewService(pluginPackageRepo, agentRepo, ipcService,
		func() pluginpkgapp.Store { return objectStoreService.Store() }, agentToken)

	sqlDB, err := db.DB()
	if err != nil {
		applog.Fatalf("get mysql pool: %v", err)
	}
	var redisPing port.PingFunc
	if redisClient != nil {
		redisPing = redisClient.Ping
	}
	mqttRepo := persistence.NewMQTTConfigRepository(db)
	infraService := infraconfigapp.NewService(
		mqttRepo, mqttbus.New(), cfg, *configPath,
		func(ctx context.Context) error { return sqlDB.PingContext(ctx) },
		redisPing, objectStoreService, cfg.UserSettings.ServerID,
	)
	eventRepo := persistence.NewEventRepository(db)
	customEventRepo := persistence.NewCustomEventRepository(db)
	customEventService := customeventapp.NewService(customEventRepo)
	eventIngest := eventingestapp.NewService(eventRepo, assetRepo, alarmRepo, objectStoreService, customEventRepo)
	infraService.SetMessageHandler(eventIngest.HandleMessage)

	eventSubRepo := persistence.NewEventSubscriptionRepository(db)
	eventSubService := eventsubapp.NewService(eventSubRepo, ipcService, pluginRepo, objectStoreService, infraService)
	agentService.SetPluginStartedHook(func(ctx context.Context, pluginID string) {
		eventSubService.RearmForPlugin(ctx, pluginID)
	})

	sipServer, err := sipinfra.NewServer(sipCfg, cfg.UserSettings.ServerID, sipCfg.Password, deviceService, redisClient)
	if err != nil {
		applog.Fatalf("init sip: %v", err)
	}
	// 库内未配 IP 时，再用媒体节点 IP / 自动探测兜底
	if sipCfg.IP == "" {
		if cfg.Media.Configured() {
			sipServer.SetLocalIP(cfg.Media.IP)
		} else if ip := firstMediaIP(cfg, mediaCluster); ip != "" {
			sipServer.SetLocalIP(ip)
		}
	}
	deviceService.SetSIP(sipServer)
	deviceService.SetCapabilities(assetRepo)
	sessionAdapter := sipinfra.NewSessionAdapter(sipServer)
	sipServer.SetRequirePreRegister(requirePreRegister)

	subordinateRepo := persistence.NewSubordinateRepository(db)
	subordinateService := downstreamapp.NewService(subordinateRepo, cfg.UserSettings.ServerID)
	sipServer.SetSubordinateHandler(subordinateService)

	alarmService := alarmapp.NewService(alarmRepo, channelRepo, deviceRepo, eventRepo, assetRepo)
	positionService := positionapp.NewService(positionRepo, channelRepo)
	sipServer.SetAlarmHandler(alarmService)
	sipServer.SetPositionHandler(positionService)

	recordTimeoutSec := cfg.UserSettings.RecordInfoTimeout / 1000
	if recordTimeoutSec <= 0 {
		recordTimeoutSec = 30
	}
	playbackService := playbackapp.NewService(
		assetRepo, deviceRepo, channelRepo, sessionAdapter, mediaCluster, cfg.UserSettings.ServerID, recordTimeoutSec,
	)
	sipServer.SetMediaHandler(playbackService)
	platformSIPClient := sipinfra.NewPlatformClient(sipCfg)
	platformService := upstreamapp.NewService(platformRepo, sipCfg, platformSIPClient, cfg.UserSettings.ServerID)
	platformChannelSvc := upstreamapp.NewChannelService(
		platformRepo, channelRepo, platformRepo, groupRegionRepo, platformSIPClient,
	)

	gbSipConfigService.SetOnChange(func(updated config.SIPConfig, requirePre bool, _ bool) {
		sipServer.ApplyConfig(updated)
		sipServer.SetRequirePreRegister(requirePre)
		deviceService.SetRequirePreRegister(requirePre)
		platformService.ApplySIPConfig(updated)
		platformSIPClient.ApplyConfig(updated)
	})

	onvifFactory := onvifinfra.NewClientFactory(30)
	onvifService := onvifapp.NewService(
		onvifDeviceRepo, onvifChannelRepo, onvifFactory, mediaCluster, cfg.UserSettings.ServerID,
	)
	playService := playapp.NewService(deviceRepo, channelRepo, sessionAdapter, mediaCluster, cfg.UserSettings.ServerID, cfg.Server.Port)
	drivers := channeldrv.New(channeldrv.Deps{
		Play: playService, ONVIF: onvifService, Plugin: ipcService,
		Device: deviceService, Devices: deviceRepo, PTZ: sessionAdapter,
	})
	ptzService := ptzapp.NewService(deviceRepo, channelRepo, assetRepo, drivers, sessionAdapter, sessionAdapter)
	sipServer.SetPTZPositionHandler(ptzService)
	sipServer.SetSnapShotHandler(snapService)
	cascadeResolver := cascadeapp.NewResolver(platformRepo, platformRepo, channelRepo)
	cascadeInbound := cascadeapp.NewInboundService(cascadeResolver, deviceRepo, playService, sessionAdapter, playbackService, sessionAdapter, sipCfg.ID, cfg.UserSettings.ServerID, platformChannelSvc)
	sipServer.SetCascadeInbound(cascadeInbound)
	// 媒体流传输模式统一由本级「国标配置」决定（标准接入不再逐设备设置）。
	streamModeProvider := func() string {
		row, err := gbSipConfigService.GetOrEmpty()
		if err != nil || row == nil {
			return ""
		}
		return row.StreamMode
	}
	playService.SetStreamModeProvider(streamModeProvider)
	playbackService.SetStreamModeProvider(streamModeProvider)
	deviceAccessService := deviceaccess.NewService(deviceService, onvifService, ipcService)
	assetService := assetapp.NewService(assetRepo, assetProjector, drivers)
	mediaSessionService := mediasessionapp.NewService(mediaServerService, assetRepo, assetService, mediaCluster, cfg.Server.Port)

	cloudRecordService := cloudrecordapp.NewService(cloudRecordRepo, mediaCluster, storageNodeService, cfg.UserSettings.ServerID)
	streamPushService := streampushapp.NewService(streamPushRepo, mediaCluster, cfg.UserSettings.ServerID)
	streamProxyService := streamproxyapp.NewService(streamProxyRepo, mediaCluster, cfg.UserSettings.ServerID)
	recordPlanService := recordplanapp.NewService(recordPlanRepo, playService, publishRegistry, storageNodeService, onvifService, cfg.UserSettings.ServerID)
	recordPlanService.Start()
	defer recordPlanService.Stop()

	gbPrefix := func() string {
		c, err := gbSipConfigService.CurrentSIP()
		if err != nil || len(c.ID) < 10 {
			return ""
		}
		return c.ID[:10]
	}
	commonChannelService := channelapp.NewService(channelRepo, groupRegionRepo, playService, playbackService, ptzService, onvifService, gbPrefix)
	groupService := groupapp.NewService(groupRegionRepo)
	regionService := regionapp.NewService(groupRegionRepo)

	statusMonitor := deviceapp.NewStatusMonitor(deviceRepo, deviceCache, cfg.UserSettings.ServerID)
	statusMonitor.Start()
	defer statusMonitor.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ops.DefaultMetrics.Start(ctx, 2*time.Second)
	infraService.Start(ctx)
	defer infraService.Stop()
	retentionDays := cfg.Event.RetentionDays
	if retentionDays == 0 {
		retentionDays = 30
	}
	eventIngest.StartRetention(ctx, retentionDays)
	// ONVIF 事件走设备侧 PullPoint 订阅，进程内直接投给事件入库，
	// 与插件那条 MQTT 链路并列。设备不支持 PullPoint 时只打日志，不影响其它能力。
	onvifService.SetEventSink(eventIngest)
	onvifService.StartEventSync(ctx)
	// 心跳：ONVIF / 插件没有主动上报的保活，靠周期探活刷新在线并写心跳时间戳，
	// 避免"设备已下线但平台仍显示在线"长期固化。
	if redisClient != nil {
		onvifService.SetKeepalivePusher(redisClient)
		ipcService.SetKeepalivePusher(redisClient)
	}
	onvifService.StartHeartbeat(ctx, 30*time.Second)
	pluginService.Start(ctx, 10*time.Second, 30*time.Second)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := ipcService.ProbeAll(ctx); err != nil {
					applog.Warnf("[plugin] 周期探活失败: %v", err)
				}
			}
		}
	}()
	agentService.Start(ctx, 10*time.Second, 30*time.Second)
	if err := sipServer.Start(ctx); err != nil {
		applog.Fatalf("start sip: %v", err)
	}
	platformService.StartEnabledPlatforms()

	mediaBaseURL := mediaServerService.FirstOnlineBaseURL()
	if mediaBaseURL == "" && cfg.Media.Configured() {
		mediaBaseURL = cfg.Media.BaseURL()
	}

	dashboard := ops.NewDashboard(persistence.NewStatsRepository(db), mediaCluster)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	router.Setup(r, router.Deps{
		AuthService:          authService,
		ONVIFService:         onvifService,
		DeviceService:        deviceService,
		DeviceAccessService:  deviceAccessService,
		PlayService:          playService,
		PlaybackService:      playbackService,
		PTZService:           ptzService,
		AlarmService:         alarmService,
		PlatformService:      platformService,
		PlatformChannelSvc:   platformChannelSvc,
		SubordinateService:   subordinateService,
		PositionService:      positionService,
		CloudRecordService:   cloudRecordService,
		StreamPushService:    streamPushService,
		StreamProxyService:   streamProxyService,
		MediaSessionService:  mediaSessionService,
		RecordPlanService:    recordPlanService,
		MediaServerService:   mediaServerService,
		StorageNodeService:   storageNodeService,
		CommonChannelSvc:     commonChannelService,
		GroupService:         groupService,
		RegionService:        regionService,
		UserRepo:             userRepo,
		PublishAuth:          publishAuth,
		StreamOnDemand:       cfg.UserSettings.StreamOnDemand,
		MediaBaseURL:         mediaBaseURL,
		JWT:                  jwtManager,
		ServerID:             cfg.UserSettings.ServerID,
		Version:              version,
		PlayTimeoutMs:        cfg.UserSettings.PlayTimeout,
		RecordInfoTimeoutMs:  cfg.UserSettings.RecordInfoTimeout,
		SIPConfig:            sipCfg,
		GbSipConfigService:   gbSipConfigService,
		ObjectStoreService:   objectStoreService,
		SnapService:          snapService,
		ServerPort:           cfg.Server.Port,
		MediaIP:              firstMediaIP(cfg, mediaCluster),
		LogDir:               applog.LogDir(cfg.Log.File),
		Metrics:              ops.DefaultMetrics,
		Dashboard:            dashboard,
		PluginService:        pluginService,
		PluginPackageService: pluginPackageService,
		IPCService:           ipcService,
		PluginToken:          pluginToken,
		AgentService:         agentService,
		AgentToken:           agentToken,
		AssetService:         assetService,
		InfraService:         infraService,
		EventSubService:      eventSubService,
		EventService:         eventIngest,
		CustomEventService:   customEventService,
		AccessTokenService:   accessTokenService,
		DictService:          dictService,
	})

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		cancel()
	}()

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	applog.Info("zero-web-server starting",
		"version", version,
		"http", addr,
		"sip_port", sipCfg.Port,
		"media_nodes", "db-managed",
	)
	if err := r.Run(addr); err != nil {
		applog.Fatalf("server exit: %v", err)
	}
}

func firstMediaIP(cfg *config.Config, media port.MediaCluster) string {
	if cfg.Media.Configured() {
		return cfg.Media.IP
	}
	if node, err := media.SelectMinimumLoad(context.Background()); err == nil {
		return node.StreamIP()
	}
	return ""
}
