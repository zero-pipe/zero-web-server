package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	appauth "zero-web-server/internal/application/auth"
	alarmapp "zero-web-server/internal/application/alarm"
	cascadeapp "zero-web-server/internal/application/cascade"
	cloudrecordapp "zero-web-server/internal/application/cloudrecord"
	channelapp "zero-web-server/internal/application/channel"
	deviceapp "zero-web-server/internal/application/device"
	deviceaccess "zero-web-server/internal/application/deviceaccess"
	groupapp "zero-web-server/internal/application/group"
	publishauth "zero-web-server/internal/application/publishauth"
	mediaserverapp "zero-web-server/internal/application/mediaserver"
	gbsipconfig "zero-web-server/internal/application/gbsipconfig"
	onvifapp "zero-web-server/internal/application/onvif"
	upstreamapp "zero-web-server/internal/application/upstream"
	downstreamapp "zero-web-server/internal/application/downstream"
	objectstoreapp "zero-web-server/internal/application/objectstore"
	snapapp "zero-web-server/internal/application/snap"
	mediacluster "zero-web-server/internal/adapter/mediacluster"
	"zero-web-server/internal/port"
	positionapp "zero-web-server/internal/application/position"
	playapp "zero-web-server/internal/application/play"
	playbackapp "zero-web-server/internal/application/playback"
	ptzapp "zero-web-server/internal/application/ptz"
	regionapp "zero-web-server/internal/application/region"
	recordplanapp "zero-web-server/internal/application/recordplan"
	streampushapp "zero-web-server/internal/application/streampush"
	streamproxyapp "zero-web-server/internal/application/streamproxy"
	"zero-web-server/internal/application/ops"
	"zero-web-server/internal/infrastructure/config"
	"zero-web-server/internal/infrastructure/civilcode"
	onvifinfra "zero-web-server/internal/infrastructure/onvif"
	"zero-web-server/internal/infrastructure/persistence"
	"zero-web-server/internal/infrastructure/persistence/mysql"
	redisinfra "zero-web-server/internal/infrastructure/redis"
	sipinfra "zero-web-server/internal/infrastructure/sip"
	"zero-web-server/internal/interfaces/http/router"
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
	groupRegionRepo := persistence.NewGroupRegionRepository(db)
	gbSipConfigRepo := persistence.NewGbSipConfigRepository(db)
	gbSipConfigService := gbsipconfig.NewService(gbSipConfigRepo, cfg.SIP)

	// 库空则用 yaml 默认 SIP 写入库；之后以库为准，页面改编码可热更新
	sipCfg, err := gbSipConfigService.Bootstrap()
	if err != nil {
		applog.Fatalf("load gb sip config: %v", err)
	}

	authService := appauth.NewService(userRepo, jwtManager, cfg.UserSettings.ServerID)
	deviceService := deviceapp.NewService(deviceRepo, channelRepo, redisClient)
	requirePreRegister := cfg.GB.RequirePreRegister
	if row, err := gbSipConfigService.GetOrEmpty(); err == nil && row != nil {
		// 库内开关优先（页面可改）；种子行默认 true
		requirePreRegister = row.RequirePreRegister
	}
	deviceService.SetRequirePreRegister(requirePreRegister)
	publishRegistry := publishauth.NewPublishRegistry()
	publishAuth := publishauth.NewPublishAuth(
		userRepo, streamPushRepo, streamProxyRepo,
		cfg.UserSettings.PushAuthority, cfg.UserSettings.RecordPushLive,
		publishRegistry,
	)
	// 媒体节点以数据库为准；启动时允许为空，由页面动态添加
	mediaServerService := mediaserverapp.NewService(mediaServerRepo, cfg.UserSettings.ServerID)
	mediaCluster := mediacluster.New(mediaServerService)

	objectStoreRepo := persistence.NewObjectStoreConfigRepository(db)
	objectStoreService := objectstoreapp.NewService(objectStoreRepo)
	snapService := snapapp.NewService(objectStoreService)

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
	sipServer.SetRequirePreRegister(requirePreRegister)

	subordinateRepo := persistence.NewSubordinateRepository(db)
	subordinateService := downstreamapp.NewService(subordinateRepo, cfg.UserSettings.ServerID)
	sipServer.SetSubordinateHandler(subordinateService)

	alarmService := alarmapp.NewService(alarmRepo, channelRepo)
	positionService := positionapp.NewService(positionRepo, channelRepo)
	sipServer.SetAlarmHandler(alarmService)
	sipServer.SetPositionHandler(positionService)

	recordTimeoutSec := cfg.UserSettings.RecordInfoTimeout / 1000
	if recordTimeoutSec <= 0 {
		recordTimeoutSec = 30
	}
	playbackService := playbackapp.NewService(
		deviceRepo, channelRepo, sipServer, mediaCluster, cfg.UserSettings.ServerID, recordTimeoutSec,
	)
	platformService := upstreamapp.NewService(platformRepo, sipCfg, cfg.UserSettings.ServerID)
	platformSIPClient := sipinfra.NewPlatformClient(sipCfg)
	platformChannelSvc := upstreamapp.NewChannelService(
		platformRepo, channelRepo, platformRepo, platformSIPClient,
	)

	gbSipConfigService.SetOnChange(func(updated config.SIPConfig, requirePre bool, _ bool) {
		sipServer.ApplyConfig(updated)
		sipServer.SetRequirePreRegister(requirePre)
		deviceService.SetRequirePreRegister(requirePre)
		platformService.ApplySIPConfig(updated)
		platformSIPClient.ApplyConfig(updated)
	})

	playService := playapp.NewService(deviceRepo, channelRepo, sipServer, mediaCluster, cfg.UserSettings.ServerID, cfg.Server.Port)
	ptzService := ptzapp.NewService(deviceRepo, sipServer)
	cascadeResolver := cascadeapp.NewResolver(platformRepo, platformRepo, channelRepo)
	cascadeInbound := cascadeapp.NewInboundService(cascadeResolver, deviceRepo, playService, sipServer)
	sipServer.SetCascadeInbound(cascadeInbound)
	onvifFactory := onvifinfra.NewClientFactory(30)
	onvifService := onvifapp.NewService(
		onvifDeviceRepo, onvifChannelRepo, onvifFactory, mediaCluster, cfg.UserSettings.ServerID,
	)
	deviceAccessService := deviceaccess.NewService(deviceService, onvifService)

	cloudRecordService := cloudrecordapp.NewService(cloudRecordRepo, mediaCluster, cfg.UserSettings.ServerID)
	streamPushService := streampushapp.NewService(streamPushRepo, mediaCluster, cfg.UserSettings.ServerID)
	streamProxyService := streamproxyapp.NewService(streamProxyRepo, mediaCluster, cfg.UserSettings.ServerID)
	recordPlanService := recordplanapp.NewService(recordPlanRepo, playService, publishRegistry, cfg.UserSettings.ServerID)
	recordPlanService.Start()
	defer recordPlanService.Stop()

	commonChannelService := channelapp.NewService(channelRepo, groupRegionRepo, playService, playbackService, ptzService, onvifService)
	groupService := groupapp.NewService(groupRegionRepo)
	regionService := regionapp.NewService(groupRegionRepo)

	statusMonitor := deviceapp.NewStatusMonitor(deviceRepo, redisClient, cfg.UserSettings.ServerID)
	statusMonitor.Start()
	defer statusMonitor.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ops.DefaultMetrics.Start(ctx, 2*time.Second)
	if err := sipServer.Start(ctx); err != nil {
		applog.Fatalf("start sip: %v", err)
	}
	platformService.StartEnabledPlatforms()

	mediaBaseURL := mediaServerService.FirstOnlineBaseURL()
	if mediaBaseURL == "" && cfg.Media.Configured() {
		mediaBaseURL = cfg.Media.BaseURL()
	}

	dashboard := ops.NewDashboard(db, mediaCluster)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	router.Setup(r, router.Deps{
		AuthService:         authService,
		ONVIFService:        onvifService,
		DeviceService:       deviceService,
		DeviceAccessService: deviceAccessService,
		PlayService:         playService,
		PlaybackService:     playbackService,
		PTZService:          ptzService,
		AlarmService:        alarmService,
		PlatformService:     platformService,
		PlatformChannelSvc:  platformChannelSvc,
		SubordinateService:  subordinateService,
		PositionService:     positionService,
		CloudRecordService:  cloudRecordService,
		StreamPushService:   streamPushService,
		StreamProxyService:  streamProxyService,
		RecordPlanService:   recordPlanService,
		MediaServerService:  mediaServerService,
		CommonChannelSvc:    commonChannelService,
		GroupService:        groupService,
		RegionService:       regionService,
		UserRepo:            userRepo,
		PublishAuth:         publishAuth,
		StreamOnDemand:      cfg.UserSettings.StreamOnDemand,
		MediaBaseURL:        mediaBaseURL,
		JWT:                 jwtManager,
		ServerID:            cfg.UserSettings.ServerID,
		Version:             version,
		PlayTimeoutMs:       cfg.UserSettings.PlayTimeout,
		RecordInfoTimeoutMs: cfg.UserSettings.RecordInfoTimeout,
		SIPConfig:           sipCfg,
		GbSipConfigService:  gbSipConfigService,
		ObjectStoreService:  objectStoreService,
		SnapService:         snapService,
		ServerPort:          cfg.Server.Port,
		MediaIP:             firstMediaIP(cfg, mediaCluster),
		LogDir:              applog.LogDir(cfg.Log.File),
		Metrics:             ops.DefaultMetrics,
		Dashboard:           dashboard,
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
