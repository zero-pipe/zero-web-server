package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	appauth "zero-web-kit/internal/application/auth"
	alarmapp "zero-web-kit/internal/application/alarm"
	cloudrecordapp "zero-web-kit/internal/application/cloudrecord"
	commonchannelapp "zero-web-kit/internal/application/commonchannel"
	deviceapp "zero-web-kit/internal/application/device"
	groupapp "zero-web-kit/internal/application/group"
	mediaapp "zero-web-kit/internal/application/media"
	mediaserverapp "zero-web-kit/internal/application/mediaserver"
	gbsipconfig "zero-web-kit/internal/application/gbsipconfig"
	onvifapp "zero-web-kit/internal/application/onvif"
	platformapp "zero-web-kit/internal/application/platform"
	positionapp "zero-web-kit/internal/application/position"
	playapp "zero-web-kit/internal/application/play"
	playbackapp "zero-web-kit/internal/application/playback"
	ptzapp "zero-web-kit/internal/application/ptz"
	regionapp "zero-web-kit/internal/application/region"
	recordplanapp "zero-web-kit/internal/application/recordplan"
	streampushapp "zero-web-kit/internal/application/streampush"
	streamproxyapp "zero-web-kit/internal/application/streamproxy"
	"zero-web-kit/internal/infrastructure/config"
	onvifinfra "zero-web-kit/internal/infrastructure/onvif"
	"zero-web-kit/internal/infrastructure/persistence"
	"zero-web-kit/internal/infrastructure/persistence/mysql"
	redisinfra "zero-web-kit/internal/infrastructure/redis"
	sipinfra "zero-web-kit/internal/infrastructure/sip"
	"zero-web-kit/internal/interfaces/http/router"
	jwtmgr "zero-web-kit/pkg/jwt"
	applog "zero-web-kit/pkg/log"

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
	if err := applog.Init(cfg.Log); err != nil {
		applog.Fatalf("init log: %v", err)
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
	gbSipConfigService := gbsipconfig.NewService(gbSipConfigRepo)

	// 国标 SIP 以数据库为准；库空则跳过监听，由「国标配置」页面填写
	sipCfg, err := gbSipConfigService.Load()
	if err != nil {
		applog.Fatalf("load gb sip config: %v", err)
	}

	authService := appauth.NewService(userRepo, jwtManager, cfg.UserSettings.ServerID)
	deviceService := deviceapp.NewService(deviceRepo, channelRepo, redisClient)
	publishRegistry := mediaapp.NewPublishRegistry()
	publishAuth := mediaapp.NewPublishAuth(
		userRepo, streamPushRepo, streamProxyRepo,
		cfg.UserSettings.PushAuthority, cfg.UserSettings.RecordPushLive,
		publishRegistry,
	)
	// 媒体节点以数据库为准；启动时允许为空，由页面动态添加
	mediaServerService := mediaserverapp.NewService(mediaServerRepo, cfg.UserSettings.ServerID)

	sipServer, err := sipinfra.NewServer(sipCfg, cfg.UserSettings.ServerID, sipCfg.Password, deviceService, redisClient)
	if err != nil {
		applog.Fatalf("init sip: %v", err)
	}
	// 库内未配 IP 时，再用媒体节点 IP / 自动探测兜底
	if sipCfg.IP == "" {
		if cfg.Media.Configured() {
			sipServer.SetLocalIP(cfg.Media.IP)
		} else if ip := firstMediaIP(cfg, mediaServerService); ip != "" {
			sipServer.SetLocalIP(ip)
		}
	}
	deviceService.SetSIP(sipServer)

	alarmService := alarmapp.NewService(alarmRepo, channelRepo)
	positionService := positionapp.NewService(positionRepo, channelRepo)
	sipServer.SetAlarmHandler(alarmService)
	sipServer.SetPositionHandler(positionService)

	recordTimeoutSec := cfg.UserSettings.RecordInfoTimeout / 1000
	if recordTimeoutSec <= 0 {
		recordTimeoutSec = 30
	}
	playbackService := playbackapp.NewService(
		deviceRepo, channelRepo, sipServer, mediaServerService, cfg.UserSettings.ServerID, recordTimeoutSec,
	)
	platformService := platformapp.NewService(platformRepo, sipCfg, cfg.UserSettings.ServerID)
	platformSIPClient := sipinfra.NewPlatformClient(sipCfg)
	platformChannelSvc := platformapp.NewChannelService(
		platformRepo, channelRepo, platformRepo, platformSIPClient,
	)

	gbSipConfigService.SetOnChange(func(updated config.SIPConfig, _ bool) {
		sipServer.ApplyConfig(updated)
		platformService.ApplySIPConfig(updated)
		platformSIPClient.ApplyConfig(updated)
	})

	playService := playapp.NewService(deviceRepo, channelRepo, sipServer, mediaServerService, cfg.UserSettings.ServerID, cfg.Server.Port)
	ptzService := ptzapp.NewService(deviceRepo, sipServer)
	onvifFactory := onvifinfra.NewClientFactory(30)
	onvifService := onvifapp.NewService(
		onvifDeviceRepo, onvifChannelRepo, onvifFactory, mediaServerService, cfg.UserSettings.ServerID,
	)

	cloudRecordService := cloudrecordapp.NewService(cloudRecordRepo, mediaServerService, cfg.UserSettings.ServerID)
	streamPushService := streampushapp.NewService(streamPushRepo, mediaServerService, cfg.UserSettings.ServerID)
	streamProxyService := streamproxyapp.NewService(streamProxyRepo, mediaServerService, cfg.UserSettings.ServerID)
	recordPlanService := recordplanapp.NewService(recordPlanRepo, playService, publishRegistry, cfg.UserSettings.ServerID)
	recordPlanService.Start()
	defer recordPlanService.Stop()

	commonChannelService := commonchannelapp.NewService(channelRepo, groupRegionRepo, playService, playbackService, ptzService, onvifService)
	groupService := groupapp.NewService(groupRegionRepo)
	regionService := regionapp.NewService(groupRegionRepo)

	statusMonitor := deviceapp.NewStatusMonitor(deviceRepo, redisClient, cfg.UserSettings.ServerID)
	statusMonitor.Start()
	defer statusMonitor.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := sipServer.Start(ctx); err != nil {
		applog.Fatalf("start sip: %v", err)
	}
	platformService.StartEnabledPlatforms()

	mediaBaseURL := mediaServerService.FirstOnlineBaseURL()
	if mediaBaseURL == "" && cfg.Media.Configured() {
		mediaBaseURL = cfg.Media.BaseURL()
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	router.Setup(r, router.Deps{
		AuthService:         authService,
		ONVIFService:        onvifService,
		DeviceService:       deviceService,
		PlayService:         playService,
		PlaybackService:     playbackService,
		PTZService:          ptzService,
		AlarmService:        alarmService,
		PlatformService:     platformService,
		PlatformChannelSvc:  platformChannelSvc,
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
		ServerPort:          cfg.Server.Port,
		MediaIP:             firstMediaIP(cfg, mediaServerService),
	})

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		cancel()
	}()

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	applog.Info("zero-web-kit starting",
		"version", version,
		"http", addr,
		"sip_port", sipCfg.Port,
		"media_nodes", "db-managed",
	)
	if err := r.Run(addr); err != nil {
		applog.Fatalf("server exit: %v", err)
	}
}

func firstMediaIP(cfg *config.Config, ms *mediaserverapp.Service) string {
	if cfg.Media.Configured() {
		return cfg.Media.IP
	}
	if node, err := ms.SelectMinimumLoad(); err == nil {
		return node.StreamIP()
	}
	return ""
}
